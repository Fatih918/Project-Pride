package zipinspect

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strings"
	"unicode/utf8"

	"pride-plugin-module/domain"
	"pride-plugin-module/installation"
)

const (
	policySchema = "urn:pride:plugin-zip-installation-policy:1"
	manifestSchema = "urn:pride:plugin-manifest:1"
	eocdSignature = uint32(0x06054b50)
	centralSignature = uint32(0x02014b50)
	localSignature = uint32(0x04034b50)
	zip64LocatorSignature = uint32(0x07064b50)
	zip64ExtraID = uint16(0x0001)
)

var (
	ErrInvalidPolicy = errors.New("invalid ZIP installation policy")
	ErrInvalidArchive = errors.New("invalid Plugin ZIP archive")
	ErrUnsupportedZIP64 = errors.New("ZIP64 is not supported by policy v1")
	ErrUnsafeEntry = errors.New("unsafe ZIP entry")
	ErrInvalidManifest = errors.New("invalid Plugin Manifest")
)

type Policy struct {
	Schema string `json:"$schema"`
	SchemaVersion string `json:"schema_version"`
	PolicyID string `json:"policy_id"`
	PackageFormat struct {
		ArchiveFormat string `json:"archive_format"`
		MediaType string `json:"media_type"`
	} `json:"package_format"`
	Limits struct {
		MaxCompressedArchiveBytes uint64 `json:"max_compressed_archive_bytes"`
		MaxTotalExpandedBytes uint64 `json:"max_total_expanded_bytes"`
		MaxSingleFileBytes uint64 `json:"max_single_file_bytes"`
		MaxEntryCount uint64 `json:"max_entry_count"`
		MaxDirectoryDepth uint64 `json:"max_directory_depth"`
		MaxCompressionRatio uint64 `json:"max_compression_ratio"`
		MaxPathLengthBytes uint64 `json:"max_path_length_bytes"`
		MaxMetadataAllocationBytes uint64 `json:"max_metadata_allocation_bytes"`
	} `json:"limits"`
}

func DefaultPolicy() Policy {
	var p Policy
	p.Schema, p.SchemaVersion, p.PolicyID = policySchema, "1.0.0", "pride.plugin.zip-installation"
	p.PackageFormat.ArchiveFormat, p.PackageFormat.MediaType = "zip", "application/vnd.pride.plugin+zip"
	p.Limits.MaxCompressedArchiveBytes = 104857600
	p.Limits.MaxTotalExpandedBytes = 536870912
	p.Limits.MaxSingleFileBytes = 134217728
	p.Limits.MaxEntryCount = 10000
	p.Limits.MaxDirectoryDepth = 32
	p.Limits.MaxCompressionRatio = 100
	p.Limits.MaxPathLengthBytes = 240
	p.Limits.MaxMetadataAllocationBytes = 16777216
	return p
}

// LoadPolicy loads the versioned policy instance. The accepted v1 values are
// exact; a missing, malformed, or modified policy fails closed.
func LoadPolicy(r io.Reader) (Policy, error) {
	raw, err := io.ReadAll(io.LimitReader(r, (1<<20)+1))
	if err != nil || len(raw) > 1<<20 {
		return Policy{}, ErrInvalidPolicy
	}
	if err := rejectDuplicateJSONKeys(raw); err != nil {
		return Policy{}, fmt.Errorf("%w: %v", ErrInvalidPolicy, err)
	}
	var p Policy
	if err := json.Unmarshal(raw, &p); err != nil {
		return Policy{}, fmt.Errorf("%w: %v", ErrInvalidPolicy, err)
	}
	if err := p.Validate(); err != nil {
		return Policy{}, err
	}
	return p, nil
}

func (p Policy) Validate() error {
	expected := DefaultPolicy()
	if p != expected { return ErrInvalidPolicy }
	return nil
}

type ArtifactSource interface {
	OpenArtifactFile(context.Context, string) (*os.File, error)
}

type Inspector struct {
	source ArtifactSource
	policy Policy
}

func New(source ArtifactSource, policy Policy) (*Inspector, error) {
	if source == nil { return nil, errors.New("ZIP artifact source is required") }
	if err := policy.Validate(); err != nil { return nil, err }
	return &Inspector{source: source, policy: policy}, nil
}

type centralEntry struct {
	name string
	normalized string
	directory bool
	compressed uint64
	expanded uint64
	localOffset uint64
	flags uint16
	method uint16
}

func (i *Inspector) Inspect(ctx context.Context, stagingPath string) (installation.InspectedPackage, error) {
	if err := ctx.Err(); err != nil { return installation.InspectedPackage{}, err }
	file, err := i.source.OpenArtifactFile(ctx, stagingPath)
	if err != nil { return installation.InspectedPackage{}, fmt.Errorf("open staged ZIP artifact: %w", err) }
	defer file.Close()
	info, err := file.Stat()
	if err != nil { return installation.InspectedPackage{}, fmt.Errorf("%w: stat archive: %v", ErrInvalidArchive, err) }
	if !info.Mode().IsRegular() || info.Size() <= 0 || uint64(info.Size()) > i.policy.Limits.MaxCompressedArchiveBytes {
		return installation.InspectedPackage{}, fmt.Errorf("%w: archive size is outside policy", ErrInvalidArchive)
	}
	entries, centralOffset, expanded, err := preflight(file, uint64(info.Size()), i.policy)
	if err != nil { return installation.InspectedPackage{}, err }
	if err := ctx.Err(); err != nil { return installation.InspectedPackage{}, err }

	reader, err := zip.NewReader(file, info.Size())
	if err != nil { return installation.InspectedPackage{}, fmt.Errorf("%w: parse ZIP directory: %v", ErrInvalidArchive, err) }
	if len(reader.File) != len(entries) { return installation.InspectedPackage{}, ErrInvalidArchive }

	var manifestBytes []byte
	files := make(map[string]struct{}, len(entries))
	var actualExpanded uint64
	for index, entry := range entries {
		if err := ctx.Err(); err != nil { return installation.InspectedPackage{}, err }
		zf := reader.File[index]
		if zf.Name != entry.name || zf.Method != entry.method || zf.Flags != entry.flags ||
			uint64(zf.CompressedSize64) != entry.compressed || uint64(zf.UncompressedSize64) != entry.expanded {
			return installation.InspectedPackage{}, fmt.Errorf("%w: central directory mismatch", ErrInvalidArchive)
		}
		files[entry.normalized] = struct{}{}
		if entry.directory { continue }
		rc, openErr := zf.Open()
		if openErr != nil { return installation.InspectedPackage{}, fmt.Errorf("%w: invalid local header for %q: %v", ErrInvalidArchive, entry.normalized, openErr) }
		var content []byte
		var actual uint64
		if entry.normalized == "plugin.json" {
			content, actual, err = readEntry(ctx, rc, true, i.policy.Limits.MaxSingleFileBytes)
			if err == nil { manifestBytes = content }
		} else {
			_, actual, err = readEntry(ctx, rc, false, i.policy.Limits.MaxSingleFileBytes)
		}
		closeErr := rc.Close()
		if err != nil { return installation.InspectedPackage{}, fmt.Errorf("%w: read entry %q: %v", ErrInvalidArchive, entry.normalized, err) }
		if closeErr != nil { return installation.InspectedPackage{}, fmt.Errorf("%w: close entry %q: %v", ErrInvalidArchive, entry.normalized, closeErr) }
		if actual != entry.expanded { return installation.InspectedPackage{}, fmt.Errorf("%w: expanded size mismatch for %q", ErrInvalidArchive, entry.normalized) }
		if actualExpanded > ^uint64(0)-actual { return installation.InspectedPackage{}, ErrInvalidArchive }
		actualExpanded += actual
		if actualExpanded > i.policy.Limits.MaxTotalExpandedBytes { return installation.InspectedPackage{}, fmt.Errorf("%w: actual expanded bytes exceed policy", ErrInvalidArchive) }
	}
	if manifestBytes == nil { return installation.InspectedPackage{}, fmt.Errorf("%w: root plugin.json is required", ErrInvalidManifest) }
	manifest, err := decodeManifest(manifestBytes)
	if err != nil { return installation.InspectedPackage{}, err }
	entryPoint, err := normalizeEntry(manifest.EntryPoint, false, i.policy)
	if err != nil { return installation.InspectedPackage{}, fmt.Errorf("%w: invalid entry_point", ErrInvalidManifest) }
	if _, ok := files["package/"+entryPoint]; !ok { return installation.InspectedPackage{}, fmt.Errorf("%w: entry_point does not exist under package/", ErrInvalidManifest) }
	_ = centralOffset
	_ = expanded
	return installation.InspectedPackage{Manifest: manifest, ManifestJSON: append([]byte(nil), manifestBytes...), EntryCount: len(entries), ExpandedBytes: actualExpanded}, nil
}

func readEntry(ctx context.Context, reader io.Reader, capture bool, maximum uint64) ([]byte, uint64, error) {
	var output bytes.Buffer
	buffer := make([]byte, 32*1024)
	var total uint64
	for {
		if err := ctx.Err(); err != nil { return nil, total, err }
		n, err := reader.Read(buffer)
		if n > 0 {
			if total > ^uint64(0)-uint64(n) || total+uint64(n) > maximum { return nil, total, ErrInvalidArchive }
			total += uint64(n)
			if capture { if _, writeErr := output.Write(buffer[:n]); writeErr != nil { return nil, total, writeErr } }
		}
		if err == io.EOF { break }
		if err != nil { return nil, total, err }
		if n == 0 { return nil, total, io.ErrNoProgress }
	}
	return output.Bytes(), total, nil
}

func preflight(file *os.File, size uint64, policy Policy) ([]centralEntry, uint64, uint64, error) {
	const maxEOCDSearch = 22 + 65535
	searchSize := size
	if searchSize > maxEOCDSearch { searchSize = maxEOCDSearch }
	tail := make([]byte, searchSize)
	if _, err := file.ReadAt(tail, int64(size-searchSize)); err != nil { return nil, 0, 0, fmt.Errorf("%w: read ZIP trailer: %v", ErrInvalidArchive, err) }
	rel := bytes.LastIndex(tail, []byte{0x50, 0x4b, 0x05, 0x06})
	if rel < 0 || len(tail)-rel < 22 { return nil, 0, 0, ErrInvalidArchive }
	eocd := tail[rel:]
	if int(binary.LittleEndian.Uint16(eocd[20:22])) != len(eocd)-22 { return nil, 0, 0, ErrInvalidArchive }
	eocdOffset := size-searchSize+uint64(rel)
	if eocdOffset >= 20 {
		var sig [4]byte
		_, _ = file.ReadAt(sig[:], int64(eocdOffset-20))
		if binary.LittleEndian.Uint32(sig[:]) == zip64LocatorSignature { return nil, 0, 0, ErrUnsupportedZIP64 }
	}
	disk := binary.LittleEndian.Uint16(eocd[4:6])
	cdDisk := binary.LittleEndian.Uint16(eocd[6:8])
	diskEntries := binary.LittleEndian.Uint16(eocd[8:10])
	totalEntries := binary.LittleEndian.Uint16(eocd[10:12])
	cdSize := uint64(binary.LittleEndian.Uint32(eocd[12:16]))
	cdOffset := uint64(binary.LittleEndian.Uint32(eocd[16:20]))
	if disk != 0 || cdDisk != 0 || diskEntries != totalEntries { return nil, 0, 0, ErrInvalidArchive }
	if totalEntries == 0xffff || cdSize == 0xffffffff || cdOffset == 0xffffffff { return nil, 0, 0, ErrUnsupportedZIP64 }
	if uint64(totalEntries) > policy.Limits.MaxEntryCount || cdSize > policy.Limits.MaxMetadataAllocationBytes { return nil, 0, 0, fmt.Errorf("%w: entry or metadata limit exceeded", ErrInvalidArchive) }
	if cdOffset+cdSize < cdOffset || cdOffset+cdSize != eocdOffset { return nil, 0, 0, ErrInvalidArchive }
	central := make([]byte, cdSize)
	if _, err := file.ReadAt(central, int64(cdOffset)); err != nil { return nil, 0, 0, ErrInvalidArchive }
	entries := make([]centralEntry, 0, totalEntries)
	seen := make(map[string]struct{}, totalEntries)
	var expanded, metadata uint64
	metadata = cdSize
	cursor := 0
	for cursor < len(central) {
		if len(central)-cursor < 46 || binary.LittleEndian.Uint32(central[cursor:cursor+4]) != centralSignature { return nil, 0, 0, ErrInvalidArchive }
		h := central[cursor:cursor+46]
		flags := binary.LittleEndian.Uint16(h[8:10])
		method := binary.LittleEndian.Uint16(h[10:12])
		compressed := uint64(binary.LittleEndian.Uint32(h[20:24]))
		uncompressed := uint64(binary.LittleEndian.Uint32(h[24:28]))
		nameLen := int(binary.LittleEndian.Uint16(h[28:30]))
		extraLen := int(binary.LittleEndian.Uint16(h[30:32]))
		commentLen := int(binary.LittleEndian.Uint16(h[32:34]))
		diskStart := binary.LittleEndian.Uint16(h[34:36])
		creatorOS := binary.LittleEndian.Uint16(h[4:6]) >> 8
		external := binary.LittleEndian.Uint32(h[38:42])
		crc32 := binary.LittleEndian.Uint32(h[16:20])
		localOffset := uint64(binary.LittleEndian.Uint32(h[42:46]))
		recordLen := 46+nameLen+extraLen+commentLen
		if recordLen > len(central)-cursor || nameLen == 0 || diskStart != 0 { return nil, 0, 0, ErrInvalidArchive }
		nameBytes := central[cursor+46:cursor+46+nameLen]
		extra := central[cursor+46+nameLen:cursor+46+nameLen+extraLen]
		if !utf8.Valid(nameBytes) || (flags&0x0800 == 0 && !isASCII(nameBytes)) { return nil, 0, 0, ErrInvalidArchive }
		if hasExtra(extra, zip64ExtraID) { return nil, 0, 0, ErrUnsupportedZIP64 }
		if compressed == 0xffffffff || uncompressed == 0xffffffff || localOffset == 0xffffffff { return nil, 0, 0, ErrUnsupportedZIP64 }
		if err := validateExtra(extra); err != nil { return nil, 0, 0, err }
		if flags&0x0001 != 0 || flags&0x0040 != 0 { return nil, 0, 0, fmt.Errorf("%w: encrypted entry", ErrInvalidArchive) }
		if method != zip.Store && method != zip.Deflate { return nil, 0, 0, fmt.Errorf("%w: unsupported compression method", ErrInvalidArchive) }
		name := string(nameBytes)
		directory := strings.HasSuffix(name, "/")
		normalized, err := normalizeEntry(name, directory, policy)
		if err != nil { return nil, 0, 0, err }
		if normalized != "plugin.json" && normalized != "package" && !strings.HasPrefix(normalized, "package/") { return nil, 0, 0, ErrUnsafeEntry }
		if normalized == "plugin.json" && directory { return nil, 0, 0, ErrUnsafeEntry }
		folded := strings.ToLower(normalized)
		if _, exists := seen[folded]; exists { return nil, 0, 0, fmt.Errorf("%w: duplicate or case-fold collision", ErrUnsafeEntry) }
		seen[folded] = struct{}{}
		modeType := (external >> 16) & 0170000
		if modeType != 0 && modeType != 0100000 && modeType != 0040000 { return nil, 0, 0, ErrUnsafeEntry }
		if external&0x00000400 != 0 || ((creatorOS == 0 || creatorOS == 10) && (external&0x10 != 0) != directory) { return nil, 0, 0, ErrUnsafeEntry }
		if modeType == 0040000 && !directory { return nil, 0, 0, ErrUnsafeEntry }
		if modeType == 0100000 && directory { return nil, 0, 0, ErrUnsafeEntry }
		if directory && (compressed != 0 || uncompressed != 0) { return nil, 0, 0, ErrUnsafeEntry }
		if !directory {
			if uncompressed > policy.Limits.MaxSingleFileBytes { return nil, 0, 0, ErrInvalidArchive }
			if compressed == 0 && uncompressed != 0 { return nil, 0, 0, ErrInvalidArchive }
			if compressed > 0 && uncompressed/compressed > policy.Limits.MaxCompressionRatio || compressed > 0 && uncompressed%compressed != 0 && uncompressed/compressed == policy.Limits.MaxCompressionRatio { return nil, 0, 0, ErrInvalidArchive }
			if expanded > ^uint64(0)-uncompressed { return nil, 0, 0, ErrInvalidArchive }
			expanded += uncompressed
			if expanded > policy.Limits.MaxTotalExpandedBytes { return nil, 0, 0, ErrInvalidArchive }
		}
		metadata += uint64(len(normalized))*2 + 128
		if metadata > policy.Limits.MaxMetadataAllocationBytes { return nil, 0, 0, ErrInvalidArchive }
		if err := validateLocalHeader(file, size, cdOffset, localOffset, nameBytes, flags, method, compressed, uncompressed, crc32); err != nil { return nil, 0, 0, err }
		entries = append(entries, centralEntry{name:name, normalized:normalized, directory:directory, compressed:compressed, expanded:uncompressed, localOffset:localOffset, flags:flags, method:method})
		cursor += recordLen
	}
	if cursor != len(central) || len(entries) != int(totalEntries) { return nil, 0, 0, ErrInvalidArchive }
	if _, ok := seen["plugin.json"]; !ok { return nil, 0, 0, ErrInvalidManifest }
	hasPackageFile := false
	for _, entry := range entries { if !entry.directory && strings.HasPrefix(entry.normalized, "package/") { hasPackageFile = true } }
	if !hasPackageFile { return nil, 0, 0, ErrInvalidArchive }
	return entries, cdOffset, expanded, nil
}

func validateLocalHeader(file *os.File, archiveSize, centralOffset, offset uint64, name []byte, flags, method uint16, compressed, expanded uint64, crc32 uint32) error {
	if offset+30 > archiveSize || offset+30 > centralOffset { return ErrInvalidArchive }
	var header [30]byte
	if _, err := file.ReadAt(header[:], int64(offset)); err != nil { return ErrInvalidArchive }
	if binary.LittleEndian.Uint32(header[:4]) != localSignature { return ErrInvalidArchive }
	localFlags, localMethod := binary.LittleEndian.Uint16(header[6:8]), binary.LittleEndian.Uint16(header[8:10])
	nameLen, extraLen := uint64(binary.LittleEndian.Uint16(header[26:28])), uint64(binary.LittleEndian.Uint16(header[28:30]))
	if localFlags != flags || localMethod != method || nameLen != uint64(len(name)) { return ErrInvalidArchive }
	localCRC := binary.LittleEndian.Uint32(header[14:18])
	localCompressed := binary.LittleEndian.Uint32(header[18:22])
	localExpanded := binary.LittleEndian.Uint32(header[22:26])
	if localCompressed == 0xffffffff || localExpanded == 0xffffffff { return ErrUnsupportedZIP64 }
	if flags&0x0008 == 0 && (localCRC != crc32 || uint64(localCompressed) != compressed || uint64(localExpanded) != expanded) { return ErrInvalidArchive }
	if flags&0x0008 != 0 && ((localCompressed != 0 && uint64(localCompressed) != compressed) || (localExpanded != 0 && uint64(localExpanded) != expanded)) { return ErrInvalidArchive }
	extraStart := offset+30+nameLen
	dataStart := extraStart+extraLen
	if dataStart < extraStart || dataStart > centralOffset { return ErrInvalidArchive }
	localName := make([]byte, nameLen)
	if _, err := file.ReadAt(localName, int64(offset+30)); err != nil || !bytes.Equal(localName,name) { return ErrInvalidArchive }
	extra := make([]byte, extraLen)
	if _, err := file.ReadAt(extra, int64(extraStart)); err != nil { return ErrInvalidArchive }
	if hasExtra(extra, zip64ExtraID) { return ErrUnsupportedZIP64 }
	if err := validateExtra(extra); err != nil { return err }
	if dataStart+compressed < dataStart || dataStart+compressed > centralOffset { return ErrInvalidArchive }
	return nil
}

func isASCII(value []byte) bool { for _, b := range value { if b >= 0x80 { return false } }; return true }

func validateExtra(extra []byte) error {
	for len(extra) > 0 {
		if len(extra) < 4 { return ErrInvalidArchive }
		length := int(binary.LittleEndian.Uint16(extra[2:4]))
		if length > len(extra)-4 { return ErrInvalidArchive }
		extra = extra[4+length:]
	}
	return nil
}

func hasExtra(extra []byte, id uint16) bool {
	for len(extra) >= 4 {
		fieldID := binary.LittleEndian.Uint16(extra[:2])
		length := int(binary.LittleEndian.Uint16(extra[2:4]))
		if length > len(extra)-4 { return false }
		if fieldID == id { return true }
		extra = extra[4+length:]
	}
	return false
}

func normalizeEntry(name string, directory bool, policy Policy) (string, error) {
	if name == "" || !utf8.ValidString(name) || strings.ContainsAny(name, "\\\x00") || strings.HasPrefix(name, "/") { return "", ErrUnsafeEntry }
	if len(name) >= 2 && ((name[0] >= 'A' && name[0] <= 'Z') || (name[0] >= 'a' && name[0] <= 'z')) && name[1] == ':' { return "", ErrUnsafeEntry }
	if directory { name = strings.TrimSuffix(name, "/") } else if strings.HasSuffix(name, "/") { return "", ErrUnsafeEntry }
	if name == "" { return "", ErrUnsafeEntry }
	parts := strings.Split(name, "/")
	for _, part := range parts { if part == "" || part == "." || part == ".." { return "", ErrUnsafeEntry } }
	clean := path.Clean(name)
	if clean != name || path.IsAbs(clean) { return "", ErrUnsafeEntry }
	depth := len(parts)
	if !directory { depth-- }
	if depth < 0 || uint64(depth) > policy.Limits.MaxDirectoryDepth || uint64(len(clean)) > policy.Limits.MaxPathLengthBytes { return "", ErrInvalidArchive }
	return clean, nil
}

type manifestDTO struct {
	Schema string `json:"$schema"`
	SchemaVersion string `json:"schema_version"`
	PluginID string `json:"plugin_id"`
	Name string `json:"name"`
	Version string `json:"version"`
	Description string `json:"description"`
	Author string `json:"author"`
	License string `json:"license"`
	EntryPoint string `json:"entry_point"`
	MinimumPRIDEVersion string `json:"minimum_pride_version"`
	Homepage string `json:"homepage"`
	DocumentationURL string `json:"documentation_url"`
	RepositoryURL string `json:"repository_url"`
	IssueTrackerURL string `json:"issue_tracker_url"`
	Keywords []string `json:"keywords"`
	Category string `json:"category"`
	Screenshots []string `json:"screenshots"`
	Icon string `json:"icon"`
	ChangelogURL string `json:"changelog_url"`
}

func decodeManifest(data []byte) (domain.Manifest, error) {
	if err := rejectDuplicateJSONKeys(data); err != nil { return domain.Manifest{}, fmt.Errorf("%w: %v", ErrInvalidManifest, err) }
	var dto manifestDTO
	if err := json.Unmarshal(data, &dto); err != nil { return domain.Manifest{}, fmt.Errorf("%w: %v", ErrInvalidManifest, err) }
	if dto.Schema != manifestSchema || dto.SchemaVersion != "1.0.0" { return domain.Manifest{}, ErrInvalidManifest }
	required := []struct{name,value string; max int}{
		{"plugin_id",dto.PluginID,255},{"name",dto.Name,255},{"version",dto.Version,128},
		{"description",dto.Description,4096},{"author",dto.Author,255},{"license",dto.License,128},
		{"entry_point",dto.EntryPoint,1024},{"minimum_pride_version",dto.MinimumPRIDEVersion,128},
	}
	for _, field := range required { if strings.TrimSpace(field.value)=="" || len([]rune(field.value))>field.max { return domain.Manifest{}, fmt.Errorf("%w: %s",ErrInvalidManifest,field.name) } }
	for _, value := range []string{dto.Homepage,dto.DocumentationURL,dto.RepositoryURL,dto.IssueTrackerURL,dto.ChangelogURL} {
		if value != "" { parsed,err:=url.Parse(value); if err!=nil || parsed.Scheme=="" { return domain.Manifest{}, ErrInvalidManifest } }
	}
	seen:=map[string]struct{}{}
	for _, keyword:=range dto.Keywords { if strings.TrimSpace(keyword)=="" { return domain.Manifest{},ErrInvalidManifest }; if _,ok:=seen[keyword];ok{return domain.Manifest{},ErrInvalidManifest};seen[keyword]=struct{}{} }
	return domain.Manifest{
		PluginID:domain.PluginID(dto.PluginID),Name:dto.Name,Version:dto.Version,Description:dto.Description,
		Author:dto.Author,License:dto.License,EntryPoint:dto.EntryPoint,MinimumPRIDEVersion:dto.MinimumPRIDEVersion,
		Homepage:dto.Homepage,DocumentationURL:dto.DocumentationURL,RepositoryURL:dto.RepositoryURL,
		IssueTrackerURL:dto.IssueTrackerURL,Keywords:append([]string(nil),dto.Keywords...),Category:dto.Category,
		Screenshots:append([]string(nil),dto.Screenshots...),Icon:dto.Icon,ChangelogURL:dto.ChangelogURL,
	},nil
}

func rejectDuplicateJSONKeys(data []byte) error {
	decoder:=json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err:=consumeJSONValue(decoder);err!=nil{return err}
	if _,err:=decoder.Token();err!=io.EOF{return errors.New("trailing JSON content")}
	return nil
}

func consumeJSONValue(decoder *json.Decoder) error {
	token,err:=decoder.Token();if err!=nil{return err}
	delim,ok:=token.(json.Delim);if !ok{return nil}
	switch delim {
	case '{':
		seen:=map[string]struct{}{}
		for decoder.More(){
			keyToken,err:=decoder.Token();if err!=nil{return err}
			key,ok:=keyToken.(string);if !ok{return ErrInvalidManifest}
			if _,exists:=seen[key];exists{return fmt.Errorf("duplicate JSON key %q",key)}
			seen[key]=struct{}{}
			if err:=consumeJSONValue(decoder);err!=nil{return err}
		}
		_,err=decoder.Token();return err
	case '[':
		for decoder.More(){if err:=consumeJSONValue(decoder);err!=nil{return err}}
		_,err=decoder.Token();return err
	default:return ErrInvalidManifest
	}
}
