package zipextract

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"pride-plugin-module/installation"
	"pride-plugin-module/installation/zipinspect"
)

var (
	ErrInvalidArchive = errors.New("invalid Plugin ZIP archive")
	ErrUnsafePath = errors.New("unsafe Plugin extraction path")
	ErrLimitExceeded = errors.New("Plugin extraction resource limit exceeded")
)

type ArtifactSource interface { OpenArtifactFile(context.Context, string) (*os.File, error) }
type Inspector interface { Inspect(context.Context, string) (installation.InspectedPackage, error) }
type Target interface {
	CreateExtractionRoot(context.Context, string) error
	CreateExtractionDirectory(context.Context, string, string) error
	CreateExtractionFile(context.Context, string, string) (*os.File, error)
	RemoveExtractionDirectory(context.Context, string, string) error
}

type Extractor struct { source ArtifactSource; inspector Inspector; target Target; policy zipinspect.Policy }

func New(source ArtifactSource, inspector Inspector, target Target, policy zipinspect.Policy) (*Extractor, error) {
	if source == nil || inspector == nil || target == nil { return nil, errors.New("ZIP extraction dependencies are required") }
	if err := policy.Validate(); err != nil { return nil, err }
	return &Extractor{source: source, inspector: inspector, target: target, policy: policy}, nil
}

// Extract revalidates the staged archive and streams only package/ payload
// files into a fresh package directory. The root plugin.json is not copied.
func (e *Extractor) Extract(ctx context.Context, stagingPath string) (string, error) {
	if err := ctx.Err(); err != nil { return "", err }
	if _, err := e.inspector.Inspect(ctx, stagingPath); err != nil {
		return "", fmt.Errorf("revalidate Plugin package before extraction: %w", err)
	}
	archive, err := e.source.OpenArtifactFile(ctx, stagingPath)
	if err != nil { return "", fmt.Errorf("open staged Plugin artifact: %w", err) }
	defer archive.Close()
	info, err := archive.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || uint64(info.Size()) > e.policy.Limits.MaxCompressedArchiveBytes {
		return "", ErrInvalidArchive
	}
	reader, err := zip.NewReader(archive, info.Size())
	if err != nil { return "", fmt.Errorf("%w: parse ZIP archive", ErrInvalidArchive) }

	const outputRoot = "package"
	if err := e.target.CreateExtractionRoot(ctx, stagingPath); err != nil {
		return "", fmt.Errorf("create fresh extraction root: %w", err)
	}
	cleanup := func(cause error) (string, error) {
		if cleanupErr := e.target.RemoveExtractionDirectory(context.WithoutCancel(ctx), stagingPath, outputRoot); cleanupErr != nil {
			return "", errors.Join(cause, fmt.Errorf("remove partial extraction: %w", cleanupErr))
		}
		return "", cause
	}

	var total uint64
	for _, entry := range reader.File {
		if err := ctx.Err(); err != nil { return cleanup(err) }
		name, directory, err := packageRelativePath(entry.Name)
		if err != nil { return cleanup(err) }
		if name == "" || name == "plugin.json" { continue }
		if directory {
			if err := e.target.CreateExtractionDirectory(ctx, stagingPath, path.Join(outputRoot, name)); err != nil {
				return cleanup(fmt.Errorf("create package directory: %w", err))
			}
			continue
		}
		if entry.Method != zip.Store && entry.Method != zip.Deflate { return cleanup(ErrInvalidArchive) }
		if entry.UncompressedSize64 > e.policy.Limits.MaxSingleFileBytes ||
			(entry.CompressedSize64 == 0 && entry.UncompressedSize64 != 0) { return cleanup(ErrLimitExceeded) }
		if entry.CompressedSize64 > 0 {
			ratio := entry.UncompressedSize64 / entry.CompressedSize64
			remainder := entry.UncompressedSize64 % entry.CompressedSize64
			if ratio > e.policy.Limits.MaxCompressionRatio || (ratio == e.policy.Limits.MaxCompressionRatio && remainder != 0) {
				return cleanup(ErrLimitExceeded)
			}
		}
		output, err := e.target.CreateExtractionFile(ctx, stagingPath, path.Join(outputRoot, name))
		if err != nil { return cleanup(fmt.Errorf("create extracted file: %w", err)) }
		input, err := entry.Open()
		if err != nil { _ = output.Close(); return cleanup(fmt.Errorf("%w: open ZIP entry", ErrInvalidArchive)) }
		written, copyErr := copyBounded(ctx, output, input, entry.UncompressedSize64,
			e.policy.Limits.MaxSingleFileBytes, &total, e.policy.Limits.MaxTotalExpandedBytes)
		inputCloseErr := input.Close()
		outputCloseErr := output.Close()
		if copyErr != nil { return cleanup(fmt.Errorf("extract package file: %w", copyErr)) }
		if inputCloseErr != nil { return cleanup(fmt.Errorf("%w: verify ZIP entry stream", ErrInvalidArchive)) }
		if outputCloseErr != nil { return cleanup(fmt.Errorf("close extracted file: %w", outputCloseErr)) }
		if written != entry.UncompressedSize64 { return cleanup(ErrInvalidArchive) }
	}
	return stagingPath + "/" + outputRoot, nil
}

func copyBounded(ctx context.Context, destination io.Writer, source io.Reader, expected, fileLimit uint64, total *uint64, totalLimit uint64) (uint64, error) {
	buffer := make([]byte, 32*1024)
	var written uint64
	for {
		if err := ctx.Err(); err != nil { return written, err }
		n, readErr := source.Read(buffer)
		if n > 0 {
			count := uint64(n)
			if count > fileLimit || count > totalLimit || count > expected ||
				written > fileLimit-count || *total > totalLimit-count || written > expected-count {
				return written, ErrLimitExceeded
			}
			w, writeErr := destination.Write(buffer[:n])
			if writeErr != nil { return written, writeErr }
			if w != n { return written, io.ErrShortWrite }
			written += count
			*total += count
		}
		if readErr == io.EOF { break }
		if readErr != nil { return written, readErr }
		if n == 0 { return written, io.ErrNoProgress }
	}
	if written != expected { return written, ErrInvalidArchive }
	return written, nil
}

func packageRelativePath(name string) (string, bool, error) {
	if name == "" || strings.Contains(name, "\\") || strings.ContainsRune(name, 0) || strings.HasPrefix(name, "/") {
		return "", false, ErrUnsafePath
	}
	directory := strings.HasSuffix(name, "/")
	trimmed := strings.TrimSuffix(name, "/")
	if trimmed == "plugin.json" && !directory { return "plugin.json", false, nil }
	if trimmed == "package" {
		if !directory { return "", false, ErrUnsafePath }
		return "", true, nil
	}
	if !strings.HasPrefix(trimmed, "package/") { return "", false, ErrUnsafePath }
	relative := strings.TrimPrefix(trimmed, "package/")
	if relative == "" || path.Clean(relative) != relative || path.IsAbs(relative) { return "", false, ErrUnsafePath }
	for _, part := range strings.Split(relative, "/") {
		if part == "" || part == "." || part == ".." { return "", false, ErrUnsafePath }
	}
	return relative, directory, nil
}
