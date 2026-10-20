Urutan implementasinya

Aku justru menyarankan:

1. Tentukan SDK Documentation Standard
↓
2. Tulis SDK Specification
↓
3. Tulis Code Standard
↓
4. Implement SDK pertama
↓
5. Tulis Usage Documentation berdasarkan API yang benar-benar ada
↓
6. Test SDK
↓
7. Baru SDK berikutnya

Dan ada satu prinsip penting:

Dokumentasi penggunaan tidak boleh mendeskripsikan API yang belum benar-benar ada.

SDK Specification

"SDK harus memiliki request, response, dan error yang terstandar."

SDK Contract

"Bentuk request-nya adalah ini, field ini wajib, tipe datanya ini, response-nya seperti ini."

Code Standard

"Implementasi SDK harus mengikuti naming, struktur kode, error handling, testing, dan sebagainya."

Documentation

"Developer memanggil SDK seperti ini."

# ROAD Map implementasi SDK

1. SDK Specification
        ↓
2. SDK Code Standard
        ↓
3. SDK Contract
        ↓
4. Implementation Design
        ↓
5. Language Selection
        ↓
6. Entry Point Design
        ↓
7. Transport / Adapter Architecture (optional)
        ↓
8. Public API Design
        ↓
9. Framework Selection (tidak dipakai sepertinya)
        ↓
10. Library / Dependency Selection
        ↓
11. Implementation
        ↓
12. SDK Documentation
        ↓
13. Contract & Implementation Test

# Algoritma implementasi SDK AI

┌─────────────────────────────────────────────────────┐
│                    Plugin (Bahasa Apapun)            │
│  (Python, Java, Go, C#, JS, Rust, dll)             │
└────────────────────────┬────────────────────────────┘
                         │
                         │  Panggil SDK Klien
                         ▼
┌─────────────────────────────────────────────────────┐
│              AI SDK (Thin Client)                   │
│  - Tersedia dalam banyak bahasa                     │
│  - Hanya kirim request ke server AI SDK            │
│  - Tidak mengandung logika AI                      │
└────────────────────────┬────────────────────────────┘
                         │
                         │  gRPC / REST / WebSocket
                         ▼
┌─────────────────────────────────────────────────────┐
│              AI SDK Server (Core)                   │
│  - Validasi request (request_id, capability, dll)  │
│  - Baca AI Configuration yang aktif                │
│  - Panggil AI Implementation (Remote/Local)        │
│  - Normalisasi response/error                      │
└─────────────────────────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────┐
│            AI Implementation (Provider)             │
│  - Remote: OpenAI, Gemini, Claude, dll             │
│  - Local: Ollama, LM Studio, llama.cpp, dll        │
└─────────────────────────────────────────────────────┘

# SDK Plugin Flow Chart

┌─────────────────────────────────────────────────────────────────────────┐
│                         PLATFORM COMPONENTS                            │
│  ┌───────────┐  ┌─────────────┐  ┌───────────────┐  ┌──────────────┐  │
│  │ Executor  │  │ Capability  │  │   Official    │  │  Marketplace │  │
│  │           │  │  Registry   │  │   Interface   │  │              │  │
│  └─────┬─────┘  └──────┬──────┘  └───────┬───────┘  └──────┬───────┘  │
│        │               │                  │                 │          │
│        └───────────────┼──────────────────┼─────────────────┘          │
│                        │                  │                            │
│                        ▼                  ▼                            │
│              ┌──────────────────────────────────────┐                  │
│              │       PLUGIN SDK (Internal)          │                  │
│              │  - Library/API dalam bahasa Core     │                  │
│              │  - Implementasi: Discovery, Lifecycle│                  │
│              │    Execution, Metadata, State        │                  │
│              └──────────────────┬───────────────────┘                  │
│                                 │                                       │
│                                 ▼                                       │
│              ┌──────────────────────────────────────┐                  │
│              │         PLUGIN MODULE (Core)         │                  │
│              │  - Mengelola Plugin yang terinstal   │                  │
│              │  - Mengelola lifecycle Plugin        │                  │
│              │  - Menyediakan Plugin Runtime        │                  │
│              └──────────────────┬───────────────────┘                  │
└─────────────────────────────────┼─────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    PLUGIN SDK (Protocol / gRPC)                        │
│  - Protokol komunikasi antara Plugin Module dan Plugin                │
│  - Definisi: .proto file (gRPC) atau OpenAPI (REST)                  │
│  - Auto-generate client untuk semua bahasa                            │
└──────────────────────────────────┬──────────────────────────────────────┘
                                   │
                                   ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                      PLUGIN IMPLEMENTATIONS                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌────────────┐│
│  │  Python      │  │    Java      │  │     Go       │  │   C#       ││
│  │  Plugin      │  │   Plugin     │  │   Plugin     │  │   Plugin   ││
│  └──────────────┘  └──────────────┘  └──────────────┘  └────────────┘│
└─────────────────────────────────────────────────────────────────────────┘

# Flow SDK Memory (Sementara)

┌─────────────────────────────────────────────────────────────────────────┐
│                         PLATFORM COMPONENTS                            │
│  ┌───────────┐  ┌─────────────┐  ┌───────────────┐  ┌──────────────┐  │
│  │  Planning │  │   Memory    │  │    Plugin     │  │  Executor    │  │
│  │           │  │  Retrieval  │  │   (Optional)  │  │              │  │
│  └─────┬─────┘  └──────┬──────┘  └───────┬───────┘  └──────┬───────┘  │
│        │               │                  │                 │          │
│        └───────────────┼──────────────────┼─────────────────┘          │
│                        │                  │                            │
│                        ▼                  ▼                            │
│              ┌──────────────────────────────────────┐                  │
│              │       MEMORY SDK (Internal)          │                  │
│              │  - Library/API dalam bahasa Core     │                  │
│              │  - Operasi: Store, Retrieve, Search, │                  │
│              │    Update, Delete, Organize, dll     │                  │
│              └──────────────────┬───────────────────┘                  │
│                                 │                                       │
│                                 ▼                                       │
│              ┌──────────────────────────────────────┐                  │
│              │         MEMORY MODULE (Core)         │                  │
│              │  - Mengelola Memory Implementations  │                  │
│              │  - Menentukan Active Implementation  │                  │
│              │  - Menerjemahkan capability → storage │                  │
│              └──────────────────┬───────────────────┘                  │
└─────────────────────────────────┼─────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    MEMORY IMPLEMENTATION                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌────────────┐│
│  │   SQLite     │  │  PostgreSQL  │  │    Vector    │  │   File     ││
│  │              │  │              │  │   Database   │  │   Storage  ││
│  └──────────────┘  └──────────────┘  └──────────────┘  └────────────┘│
└─────────────────────────────────────────────────────────────────────────┘

# STructure kebutuhan AI SDK Server

AI SDK Server
│
├── Go standard library
│   ├── context
│   ├── errors
│   ├── log/slog
│   ├── encoding/json
│   ├── crypto/tls
│   └── testing
│
├── gRPC
│   ├── google.golang.org/grpc
│   └── google.golang.org/protobuf
│
└── Provider/Configuration dependencies
    └── ditambahkan hanya jika benar-benar diperlukan

# Server flow

Implementasi server nantinya kira-kira:

gRPC Request
      │
      ▼
┌───────────────┐
│ gRPC Server   │
└───────┬───────┘
        ▼
Request Validation
        │
        ▼
AI Configuration Resolver
        │
        ▼
Active AI Configuration
        │
        ▼
AI Execution
        │
        ▼
Response Normalization
        │
        ▼
gRPC Response

# base line SDK Server

1. Dependency utama

Untuk AI SDK Server, aku sarankan:

AI SDK Server
│
├── Go standard library
│   ├── context
│   ├── errors
│   ├── log/slog
│   ├── encoding/json
│   ├── crypto/tls
│   └── testing
│
├── gRPC
│   ├── google.golang.org/grpc
│   └── google.golang.org/protobuf
│
└── Provider/Configuration dependencies
    └── ditambahkan hanya jika benar-benar diperlukan

Jadi tidak ada framework web, dependency-injection framework, ORM, atau library konfigurasi besar pada baseline.

2. Tool untuk Protobuf + gRPC

Kita membutuhkan toolchain Protobuf.

protoc

Ini adalah compiler Protocol Buffers.

Fungsinya:

ai_sdk.proto
      │
      ▼
    protoc
      │
      ├── Go message types
      └── Go gRPC service definitions
Go protobuf plugin

Kita membutuhkan:

protoc-gen-go

untuk menghasilkan kode message protobuf.

Go gRPC plugin

Dan:

protoc-gen-go-grpc

untuk menghasilkan interface/service gRPC.

Jadi toolchain-nya:

.proto
  │
  ▼
protoc
  │
  ├── protoc-gen-go
  │
  └── protoc-gen-go-grpc
          │
          ▼
Generated Go Code

Library runtime yang digunakan oleh server kemudian:

google.golang.org/grpc
google.golang.org/protobuf
3. Konsep implementasinya

Aku akan memisahkan contract, protocol, dan server implementation.

SDK Contract
     │
     ▼
AI SDK Protocol
(.proto)
     │
     ▼
Generated Code
     │
     ▼
AI SDK Server

Misalnya secara konseptual .proto nanti mempunyai service seperti:

AIService


└── Execute()

dan request:

AIRequest
├── request_id
├── capability
├── input
└── metadata

response:

AIResponse
├── request_id
├── capability
├── output
└── metadata

error mengikuti error model yang sudah kita tetapkan dalam contract.

Tapi kita belum menulis field finalnya sekarang. Itu nanti kita lakukan ketika membuat AI SDK Protocol Specification, supaya .proto benar-benar berasal dari Contract, bukan kita mengarang contract baru dari kode.

4. Server flow

Implementasi server nantinya kira-kira:

gRPC Request
      │
      ▼
┌───────────────┐
│ gRPC Server   │
└───────┬───────┘
        ▼
Request Validation
        │
        ▼
AI Configuration Resolver
        │
        ▼
Active AI Configuration
        │
        ▼
AI Execution
        │
        ▼
Response Normalization
        │
        ▼
gRPC Response

Perhatikan bahwa AI SDK Server tidak perlu tahu provider secara hard-coded.

Ia mendapatkan konfigurasi aktif, kemudian menggunakan mekanisme eksekusi yang sesuai dengan konfigurasi tersebut.

5. Struktur source code

Untuk awal, aku justru akan membuatnya sederhana:

ai-sdk-server/
│
├── proto/
│   └── ai_sdk.proto
│
├── gen/
│   └── ... generated protobuf/grpc code
│
├── cmd/
│   └── ai-sdk-server/
│       └── main.go
│
├── internal/
│   ├── server/
│   ├── config/
│   ├── execution/
│   └── validation/
│
├── go.mod
├── go.sum
└── README.md

Tidak perlu langsung membuat 30 package.

Kalau sebuah abstraction belum mempunyai alasan nyata, jangan dibuat dulu.

6. Client SDK

Client SDK nantinya dihasilkan dari .proto.

Contohnya:

                    ai_sdk.proto
                         │
             ┌───────────┼───────────┐
             ▼           ▼           ▼
          Python        Go          Rust
          Client       Client       Client
             │           │           │
             └───────────┼───────────┘
                         ▼
                  AI SDK Server

Dengan demikian bahasa berbeda tetap menggunakan contract/protocol yang sama.

Client tidak mengimplementasikan AI.

Client hanya:

Create Request
      ↓
Serialize
      ↓
gRPC
      ↓
Receive Response
      ↓
Deserialize
7. Dependency policy

Ini menurutku harus kita lock sebagai prinsip:

Wajib
Go
gRPC
Protocol Buffers
Gunakan standard library jika sudah cukup
HTTP      → net/http
Logging   → log/slog
JSON      → encoding/json
TLS       → crypto/tls
Testing   → testing
Context   → context
Errors    → errors
Jangan digunakan tanpa kebutuhan
❌ Web framework
❌ ORM
❌ DI framework
❌ Configuration framework besar
❌ Logging framework tambahan
❌ Provider SDK di core
❌ Utility library hanya untuk convenience
8. Toolchain lengkap

Jadi environment development kita kira-kira:

Go
 │
 ├── go
 │
 ├── protoc
 │
 ├── protoc-gen-go
 │
 └── protoc-gen-go-grpc

Build:

.proto
   ↓
protoc
   ↓
generated Go code
   ↓
go build
   ↓
AI SDK Server binary

Testing:

go test

Dan dependency management:

go mod

Itu sudah cukup untuk membangun fondasi server.

# Structure Source Code SDK Server AI

ai-sdk-server/
│
├── proto/
│   └── ai_sdk.proto
│
├── gen/
│   └── ... generated protobuf/grpc code
│
├── cmd/
│   └── ai-sdk-server/
│       └── main.go
│
├── internal/
│   ├── server/
│   ├── config/
│   ├── execution/
│   └── validation/
│
├── go.mod
├── go.sum
└── README.md

# keputusan design untuk .proto
Bisa. Setelah ditentukan sekarang, aku akan mengunci keputusan desain berikut untuk .proto:

Bagian	Keputusan
request_id	string
capability	string
input	google.protobuf.Any
output	google.protobuf.Any
Metadata	google.protobuf.Struct untuk metadata tambahan yang memang dinamis
Error transport	native gRPC status, bukan AIError sebagai response biasa
Error detail	google.rpc.Status / structured details bila diperlukan
Version	pride.ai.v1
RPC	Execute unary
State	Stateless
Provider config	Tidak masuk .proto
Provider-specific schema	Tidak masuk .proto

Alasan utama aku mengubah bytes → google.protobuf.Any: Any memang dirancang untuk membawa serialized protobuf message beserta identitas tipenya, sehingga kita tetap mendapat payload yang typed dan bisa dikembangkan lintas bahasa tanpa membuat root ExecuteRequest berisi schema semua capability.

Untuk error, kita tidak perlu membuat AIError sebagai return message dari Execute. gRPC sendiri sudah mempunyai status/error model lintas bahasa; error aplikasi bisa menggunakan status tersebut dan, bila diperlukan, structured protobuf error details.

Jadi bentuk finalnya kira-kira:

syntax = "proto3";

package pride.ai.v1;

import "google/protobuf/any.proto";
import "google/protobuf/struct.proto";

service AIService {
  rpc Execute(ExecuteRequest) returns (ExecuteResponse);
}

message ExecuteRequest {
  string request_id = 1;
  string capability = 2;
  google.protobuf.Any input = 3;
  google.protobuf.Struct metadata = 4;
}

message ExecuteResponse {
  string request_id = 1;
  string capability = 2;
  google.protobuf.Any output = 3;
  google.protobuf.Struct metadata = 4;
}

Ini sudah jauh lebih tepat daripada draft sebelumnya.

Satu catatan: Any berarti setiap capability yang ingin memakai payload typed harus memiliki message protobuf yang bisa di-pack ke dalam Any. Itu justru cocok dengan arsitektur kita karena semantics capability tidak perlu ditanam ke dalam AIService.

Dengan keputusan ini, enam poin yang tadi belum locked sudah bisa dianggap locked, sehingga kita bisa lanjut membuat canonical ai_sdk.proto.

# keputusan SDK server AI
Step 6 — Execute() Lifecycle & Error Flow

Kita lock flow-nya sesederhana mungkin:

ExecuteRequest
      ↓
[1] Validate
      ↓
[2] Resolve AI Configuration
      ↓
[3] Execute AI
      ↓
[4] Normalize Response
      ↓
ExecuteResponse
1. Validation Failure

Request ditolak sebelum AI dipanggil jika:

request_id kosong/tidak valid
capability kosong/tidak valid
input tidak valid

→ gRPC INVALID_ARGUMENT

2. Configuration Failure

Jika AI Configuration tidak tersedia atau tidak valid:

AI Configuration
      ↓
     FAIL

→ gRPC FAILED_PRECONDITION atau UNAVAILABLE tergantung kondisi.

Prinsipnya: jangan mengirim request ke provider jika configuration belum siap.

3. AI Execution Failure

Jika provider/runtime gagal:

Kondisi	gRPC status
Authentication gagal	UNAUTHENTICATED
Permission ditolak	PERMISSION_DENIED
Provider tidak tersedia	UNAVAILABLE
Timeout	DEADLINE_EXCEEDED
Rate limit	RESOURCE_EXHAUSTED
Capability tidak didukung	UNIMPLEMENTED
Error internal	INTERNAL

AI SDK Server melakukan normalization, sehingga consumer tidak perlu memahami error asli masing-masing provider.

4. Successful Execution
Request
  ↓
Validation ✓
  ↓
Configuration ✓
  ↓
AI Execution ✓
  ↓
Normalization ✓
  ↓
Response

Response harus mempertahankan:

request_id
capability
output
metadata
5. Tidak Ada AI Fallback di Dalam SDK Server

Ini penting.

AI SDK Server tidak menentukan apakah Core harus memakai AI atau tidak.

Core yang memutuskan:

Core
 ↓
"AI diperlukan"
 ↓
AI SDK

AI SDK Server hanya menjalankan request yang diberikan kepadanya.

Jadi kita tidak mencampurkan AI reasoning policy ke dalam SDK Server.

Step 6 selesai.

Berikutnya Step 7 — menentukan AI Configuration boundary secara konkret, yaitu apa yang boleh diketahui AI SDK Server dan apa yang tetap berada di luar boundary-nya.
Jadi Step 7 seharusnya bukan mendesain ulang AI Configuration, tetapi menentukan boundary antara AI SDK Server dan AI Configuration.

Step 7 — AI Configuration Boundary

Kita lock:

AI SDK Server
      │
      │ read active configuration
      ▼
AI Configuration
      │
      ├── implementation reference
      ├── execution parameters
      └── authentication information
              │
              ▼
       AI Implementation

AI SDK Server boleh:

membaca Active AI Configuration
menggunakan implementation reference
menggunakan execution information
menggunakan credential secara temporer saat execution

AI SDK Server tidak boleh:

membuat configuration
mengubah configuration
menghapus configuration
mengganti active configuration
menyimpan credential secara permanen
memilih provider/model secara independen dari configuration

Ini langsung konsisten dengan Contract bagian Configuration Boundary dan Security Contract.

Flow final
ExecuteRequest
      ↓
AI SDK Server
      ↓
Validate
      ↓
Read Active AI Configuration
      ↓
Resolve Implementation
      ↓
Execute Capability
      ↓
Normalize Result
      ↓
ExecuteResponse

Dan yang paling penting:

AI SDK Server tidak menjadi pemilik AI Configuration.

Ia hanya consumer/executor dari configuration yang sudah ditentukan oleh platform.

Ini juga konsisten dengan prinsip utama contract:

Request capabilities, not implementations.

Step 7 selesai. Langkah berikutnya baru masuk ke Step 8 — menentukan internal interface antara AI SDK Server dan AI Configuration/AI Execution, tanpa memasukkan detail provider ke gRPC protocol.
Step 8 — Internal AI Execution Interface

Sekarang kita tentukan interface internal antara AI SDK Server dan AI Configuration/AI Execution.

Yang kita lock adalah boundary, bukan provider API.

gRPC Client
    ↓
AI SDK Server
    ↓
AI Execution Interface
    ↓
Active AI Configuration
    ↓
Configured AI Implementation
1. AI Execution Interface

Server membutuhkan satu interface internal:

type AIExecutor interface {
    Execute(ctx context.Context, request AIExecutionRequest) (AIExecutionResult, error)
}

AI SDK Server hanya mengetahui interface tersebut.

Ia tidak mengetahui apakah di belakangnya:

OpenAI
Gemini
Claude
Ollama
llama.cpp
LM Studio
atau lainnya
2. Internal Request

Secara konseptual:

type AIExecutionRequest struct {
    Capability string
    Input      any
    Metadata   map[string]any
}

request_id tetap berada di execution context/server layer dan tidak perlu dipaksa menjadi bagian dari provider-facing request.

3. Internal Result
type AIExecutionResult struct {
    Output   any
    Metadata map[string]any
}

Server kemudian mengubahnya menjadi:

AIExecutionResult
       ↓
Response Normalizer
       ↓
ExecuteResponse
4. AI Configuration sebagai Resolver

Kita tidak membuat provider selection di AIExecutor.

Flow-nya:

ExecuteRequest
      ↓
AI SDK Server
      ↓
Configuration Resolver
      ↓
Active AI Configuration
      ↓
AIExecutor

Artinya:

Configuration menentukan execution environment.

AIExecutor hanya menjalankan execution yang sudah ditentukan.

5. Tidak Ada Provider Interface di Protocol

Kita tidak memasukkan sesuatu seperti:

type OpenAIExecutor interface {}
type GeminiExecutor interface {}
type OllamaExecutor interface {}

ke dalam architectural contract AI SDK Server.

Implementasi tersebut berada di bawah execution boundary.

6. Final Boundary
             AI SDK Server
                   │
        ┌──────────┴──────────┐
        │                     │
 Validation             Configuration
        │                     │
        └──────────┬──────────┘
                   ↓
              AIExecutor
                   ↓
        Configured AI Runtime
                   ↓
               Provider

Dengan ini server tetap tipis, tetapi bukan berarti dumb proxy. Ia tetap bertanggung jawab atas validation, configuration consumption, execution orchestration, dan response normalization.

Step 8 selesai.

Berikutnya: Step 9 — menentukan package/module boundary Go dan dependency graph, sebelum kita mulai menulis kode server.
Kita lanjut dari Step 9, karena Step 8 sudah selesai.

Step 9 — Go Module & Package Boundary

Tujuan step ini bukan membuat kode server dulu, tetapi menentukan bagaimana source Go AI SDK Server dibagi supaya tidak berubah menjadi framework besar.

Struktur konseptual yang aku sarankan:

AI SDK Server
│
├── transport/
│   └── grpc/
│
├── service/
│   └── ai/
│
├── execution/
│   └── ...
│
├── configuration/
│   └── ...
│
└── proto-generated/
Boundary-nya
gRPC
  ↓
AI Service
  ↓
Execution
  ↓
Configuration
  ↓
AI Runtime
transport

Hanya menangani:

gRPC request/response
mapping protobuf ↔ internal types
gRPC status

Tidak boleh berisi logika AI.

service

Menangani use-case AI SDK:

Execute(...)

Di sinilah request SDK diterima dan orchestration dilakukan.

execution

Berisi abstraction internal seperti:

type AIExecutor interface {
    Execute(
        ctx context.Context,
        request AIExecutionRequest,
    ) (AIExecutionResult, error)
}

Tidak mengetahui detail gRPC.

configuration

Bertanggung jawab terhadap consumption Active AI Configuration.

Bukan pemilik configuration.

Tidak boleh:

Create Configuration
Update Configuration
Switch Configuration
Persist Credentials
Generated Protobuf

Generated code tetap terpisah dari business logic:

.proto
   ↓
protoc
   ↓
generated Go

Kode generated tidak diedit manual.

Dependency Direction

Ini yang paling penting:

transport
    ↓
service
    ↓
execution
    ↓
configuration

Bukan:

configuration → gRPC
execution → gRPC
service → provider SDK

Dengan begitu kalau suatu hari transport berubah, misalnya:

gRPC
 ↓
???

bagian execution dan service tidak harus dibongkar.

Step 9 Decision

Kita lock:

Go standard packages sebagai baseline.
google.golang.org/grpc untuk transport.
google.golang.org/protobuf untuk Protobuf.
Tidak menggunakan web framework.
Tidak menggunakan dependency-injection framework.
Tidak membuat monolithic AIService yang berisi seluruh logic.
Generated protobuf code dipisahkan dari domain/server logic.
Internal execution menggunakan interface.
AI Configuration tetap boundary terpisah.

Step 9 selesai.

Step 10 berikutnya

Kita masuk ke server bootstrap & lifecycle:

main
 ↓
load configuration
 ↓
create dependencies
 ↓
create AI service
 ↓
create gRPC server
 ↓
register AIService
 ↓
listen
 ↓
graceful shutdown

Di sini baru kita mulai mendesain server yang benar-benar bisa dijalankan, tetapi belum masuk implementasi provider AI.
Step 15 — Testing Architecture

Sekarang kita menentukan bagaimana AI SDK Server akan diuji sebelum implementasi konkret dibuat.

Prinsipnya: test boundary, bukan provider.

1. Test Layers
                    AI SDK Server
                         │
          ┌──────────────┼──────────────┐
          ▼              ▼              ▼
      Unit Test     Integration Test   Proto Test
          │              │              │
          ▼              ▼              ▼
     Service/        gRPC flow       .proto
     Executor        end-to-end      compatibility
2. Unit Test

Fokus pada internal behavior.

Yang diuji:

AIService
├── validation
├── configuration resolution
├── executor invocation
└── response/error normalization

AIExecutor menggunakan fake/stub, bukan provider nyata.

Contoh:

AIService
    ↓
FakeAIExecutor
    ↓
Fake Result

Dengan ini test cepat dan deterministic.

3. Configuration Test

AIConfigurationProvider juga dibuat fake:

FakeConfigurationProvider
          ↓
Active Configuration

Kita dapat menguji:

configuration tersedia
configuration tidak tersedia
configuration invalid
configuration error

tanpa database atau service eksternal.

4. gRPC Integration Test

Kemudian kita test jalur sebenarnya:

Test Client
    ↓
gRPC
    ↓
AIService
    ↓
FakeAIExecutor
    ↓
Response

Yang diuji:

protobuf serialization
gRPC request
response
status/error
metadata
cancellation/deadline

Provider AI tetap tidak diperlukan.

5. Protocol Test

.proto juga perlu diverifikasi.

Minimal:

ai_sdk.proto
    ↓
protoc
    ↓
generated code
    ↓
compile

Test ini memastikan perubahan schema tidak menghasilkan generated code yang rusak.

6. Provider Integration Test

Ini bukan bagian dari core AI SDK Server test suite.

Nanti jika kita benar-benar mengimplementasikan AI Configuration + execution mechanism:

AI SDK Server
      ↓
AI Configuration
      ↓
Real AI Runtime

baru dibuat integration test khusus untuk execution implementation tersebut.

Jadi kegagalan provider tidak membuat protocol/service test ikut gagal.

7. Test Contract

Kita lock bahwa setiap implementation harus mempertahankan behavior berikut:

Valid Request
    → Execute
    → Standardized Response

Invalid Request
    → gRPC Error

Configuration Failure
    → gRPC Error

Execution Failure
    → Normalized gRPC Error

Cancellation
    → Execution cancellation

Deadline
    → Execution timeout/cancellation
8. Tidak Menggunakan Heavy Testing Framework

Untuk awal:

Go testing package

sudah cukup.

Tidak perlu:

❌ test framework besar
❌ mocking framework
❌ integration framework

Kita bisa menggunakan interface + fake implementation sederhana.

Ini konsisten dengan keputusan sebelumnya: dependency hanya ditambahkan jika memang diperlukan.

Step 15 Decision

🔒 Kita lock:

Unit test untuk service/execution behavior.
Fake configuration provider.
Fake AI executor.
gRPC integration test.
.proto compilation/compatibility validation.
Provider integration test dipisahkan.
Go standard testing sebagai baseline.
Tidak ada testing framework tambahan.
Sekarang fase desain server selesai.

Urutannya menjadi:

Contract
   ↓
Protocol Design
   ↓
.proto
   ↓
Toolchain
   ↓
Server Architecture
   ↓
Server Specification
   ↓
Testing Architecture
   ↓
IMPLEMENTATION

Langkah berikutnya sudah bukan sekadar desain lagi: kita bisa mulai membuat Go module + generated protobuf + skeleton AI SDK Server.

# skeleton SDKs AI
sdks/
└── ai/
    ├── go.mod
    ├── go.sum
    │
    ├── proto/
    │   ├── ai_sdk.proto
    │   ├── ai_sdk.pb.go
    │   └── ai_sdk_grpc.pb.go
    │
    └── server/
        ├── cmd/
        │   └── ai-sdk-server/
        │       └── main.go
        │
        └── internal/
            ├── service/
            │   └── ai/
            │       └── service.go
            │
            ├── configuration/
            │   ├── provider.go
            │   └── types.go
            │
            ├── implementation/
            │   ├── resolver.go
            │   └── implementation.go
            │
            └── execution/
                ├── executor.go
                └── types.go

# flow Config
User
 │
 │ input API Key melalui Official Interface
 ▼
Credential / Secret Service
 │
 │ encrypt + store
 ▼
Secret Store
 │
 │ menghasilkan credential reference
 ▼
AI Configuration
 │
 ├── ImplementationRef
 ├── ConnectionParameters
 └── CredentialRef
       │
       ▼
AI SDK Server
       │
       │ resolve CredentialRef saat execution
       ▼
AI Implementation

# roadmap implementasi SDK AI
11.1  AIConfiguration boundary              ✅
11.2  Credential Store contract              ✅
11.3  MemoryStore                            ✅
11.4  Encryption boundary                    ✅
11.5  Encryptor contract                     ✅
11.6  AES-256-GCM                            ✅
11.7  AES tests                              ✅
11.8  EncryptedStore                         ✅
11.9  SQLite persistent backend              ✅
11.9.7 SQLite + Encryption                   ✅
11.9.8 KeySource contract                    ✅
11.9.9 EnvironmentKeySource (DEV)            ✅

11.9.10 Composition / Factory                ⬅️ NEXT
11.9.11 Database lifecycle                  ⬅️
11.9.12 CredentialRef generation            ⬅️
11.9.13 Credential domain service            ⬅️
11.9.14 Credential rotation                 ⬅️
11.9.15 Credential deletion policy          ⬅️
11.9.16 AIConfiguration integration         ⬅️
11.9.17 Production KeySource                ⬅️
11.9.18 Security policy/tests               ⬅️
11.9.19 End-to-end credential flow          ⬅️

# structure boundary SDK AI paling new
                         PRIDE CORE
┌──────────────────────────────────────────────────────────┐
│                                                          │
│  Planning                                               │
│      │                                                   │
│      ▼                                                   │
│  Task Tree                                               │
│      │                                                   │
│      ▼                                                   │
│  Capability Registry                                     │
│      │                                                   │
│      ▼                                                   │
│  Execution Plan                                          │
│      │                                                   │
│      ▼                                                   │
│  Executor                                                │
│      │                                                   │
│      ▼                                                   │
│  AI Capability                                           │
│      │                                                   │
│      │ AI SDK Contract                                   │
│      ▼                                                   │
│  ┌────────────────────────────────────────────────────┐  │
│  │                    AI SDK                          │  │
│  │                                                    │  │
│  │ Request / Response / Error                         │  │
│  │                                                    │  │
│  └──────────────────────┬─────────────────────────────┘  │
│                         │                                │
│                         ▼                                │
│  ┌────────────────────────────────────────────────────┐  │
│  │                    AI MODULE                       │  │
│  │                                                    │  │
│  │  AI Configuration                                  │  │
│  │       │                                            │  │
│  │       ├── ImplementationRef                        │  │
│  │       └── CredentialRef                             │  │
│  │                                                    │  │
│  │  AI Implementation Management                      │  │
│  │                                                    │  │
│  └──────────────────────┬─────────────────────────────┘  │
│                         │                                │
└─────────────────────────┼────────────────────────────────┘
                          │
                          ▼
               ┌──────────────────────┐
               │ AI IMPLEMENTATION    │
               │                      │
               │ Remote Integration   │
               │ Local Runtime        │
               └──────────┬───────────┘
                          │
                          ▼
                 External AI System



User
 │
 ▼
Planning
 │
 ▼
Task Tree
 │
 ▼
Capability Registry
 │
 ▼
Execution Plan
 │
 ▼
Executor
 │
 ▼
AI Capability: generation
 │
 ▼
AI SDK
 │
 │ ExecuteRequest
 │ {
 │   request_id
 │   capability
 │   input
 │   metadata
 │ }
 ▼
AI Module
 │
 ├── Active AI Configuration
 │       │
 │       ├── ImplementationRef
 │       └── CredentialRef
 │
 ├── Resolve credential
 │
 └── Resolve implementation
 │
 ▼
AI Implementation
 │
 ▼
External AI
 │
 ▼
AI Implementation
 │
 ▼
AI Module
 │
 ▼
AI SDK
 │
 │ ExecuteResponse
 ▼
AI Capability
 │
 ▼
Executor
 │
 ▼
Task Completed

Boundary yang harus kita pertahankan

AI SDK memiliki:

ExecuteRequest
ExecuteResponse
AIError


Validation
Transport
SDK Contract

AI Module memiliki:

AI Configuration
CredentialRef
ImplementationRef
Active Configuration
Implementation Resolver
Credential Resolver
AI Execution

AI Implementation memiliki:

Provider/Runtime adapter
Provider API
Local runtime API
Request translation
Response normalization

Dan Core hanya mengetahui AI SDK, bukan AI Module internal.

# definisi tugas
configuration = data/definisi configuration.
selection = configuration mana yang sedang dipilih.
execution = menjalankan configuration tersebut.
implementation = implementasi/provider yang ditunjuk configuration.

# tahap pengerjaan step 12
12.1  Selection Contract
       ↓
12.2  Selection Store
       ↓
12.3  Selection Service
       ↓
12.4  Selection → Execution Integration
       ↓
12.5  Provider Adapter Contract
       ↓
12.6  Adapter Resolver
       ↓
12.7  Test Adapter
       ↓
12.8  Executor → Adapter Integration
       ↓
12.9  Provider Switching Test
       ↓
12.10 gRPC Integration Test

# step 13 - pengerjaan
Step 13 — Real AI Configuration & Adapter Integration

13.1  Configuration Creation Contract          
13.2  User-defined Configuration                
13.3  Credential Creation Boundary              
13.4  ConfigurationRef / CredentialRef         
13.5  Real Adapter Contract                     
13.6  Real Provider Integration       + Adapter architecture
+ adapter contracts
+ Gemini adapter
+ OpenAI adapter
+ Local adapter
+ deterministic adapter tests                    
13.7  Credential Existence Validation            
13.8  Real Credential Authentication             
13.9  Explicit Enable Lifecycle                  
13.10 Explicit Selection Lifecycle               
13.11 Real Provider Execution                    ⬜
13.12 Invalid Credential → AUTHENTICATION_ERROR  ⬜
13.13 Invalid Configuration rejection            ⬜
13.14 SDK remains provider-agnostic              ⬜
13.15 Full SDK → Module → Adapter → Provider E2E ⬜

tambahan lifecycle/cache abstraction berfungsi untuk menjawab pertanyaan 
    kapan credential berubah?
    kapan client lama dibuang?
    bagaimana configuration update?
    bagaimana concurrent access?
    apakah client boleh dishare antar configuration?
    bagaimana credential rotation?
    apakah CredentialRef yang sama bisa dipakai beberapa configuration?

# Refactor sementara
## Phase R1 — Canonical Error Boundary
R1.1 — Tetapkan errors.Error sebagai satu-satunya semantic error

Semua internal layer menggunakan:

*errors.Error

jika error perlu semantic classification.

Tidak ada taxonomy kedua.

R1.2 — Refactor Executor error

Ubah:

fmt.Errorf(...)

menjadi canonical error di boundary yang relevan.

Contoh:

selection failure
→ CONFIGURATION_ERROR

configuration failure
→ CONFIGURATION_ERROR

disabled configuration
→ CONFIGURATION_ERROR

implementation resolution failure
→ CONFIGURATION_ERROR

credential resolution failure
→ CONFIGURATION_ERROR

Underlying error tetap:

.WithCause(err)

untuk internal debugging.

R1.3 — Context errors

Normalisasi:

context.DeadlineExceeded
→ TIMEOUT_ERROR

context.Canceled
→ TIMEOUT_ERROR

dengan semantic yang konsisten.


## Phase R2 — Delete Duplicate gRPC Error Mapper

Ini blocker utama.

Hapus konsep:

mapExecutionError()
sanitizeExecutionError()
codeForAIError()

dari transport sebagai taxonomy kedua.

Gunakan:

prideerrors.GRPCStatus(err)

sebagai satu-satunya conversion:

PRIDE Error
 ↓
gRPC status
 ↓
AIError detail

Validation error juga masuk canonical error.

Target server:

Execute()
 ├── validate
 │     └── PRIDE Error
 │
 ├── mapper
 │     └── PRIDE Error
 │
 ├── executor
 │     └── PRIDE Error
 │
 └── success
## Phase R3 — Error Test Matrix

Buat test matrix lengkap.

Failure	PRIDE Code	gRPC
missing request ID	INVALID_REQUEST	InvalidArgument
missing capability	INVALID_REQUEST	InvalidArgument
missing input	INVALID_REQUEST	InvalidArgument
no selection	CONFIGURATION_ERROR	FailedPrecondition
config missing	CONFIGURATION_ERROR	FailedPrecondition
config disabled	CONFIGURATION_ERROR	FailedPrecondition
implementation missing	CONFIGURATION_ERROR	FailedPrecondition
credential missing	CONFIGURATION_ERROR	FailedPrecondition
invalid API key	AUTHENTICATION_ERROR	Unauthenticated
permission denied	PERMISSION_ERROR	PermissionDenied
provider unavailable	CONNECTION_ERROR	Unavailable
timeout	TIMEOUT_ERROR	DeadlineExceeded
unsupported capability	CAPABILITY_UNSUPPORTED	Unimplemented
rate limit	RATE_LIMIT	ResourceExhausted
unexpected provider error	RUNTIME_ERROR	Internal
unknown error	UNKNOWN_ERROR	Unknown

Ini akan menjadi salah satu golden contracts PRIDE.

## Phase R4 — Credential Optionality

Refactor:

CredentialRef == ""

menjadi valid untuk execution.

Executor:

if config.CredentialRef != "" {
    resolve
} else {
    Credential = nil
}

Kemudian test:

Local/no-auth configuration
→ Execute succeeds

dan:

CredentialRef exists but credential missing
→ CONFIGURATION_ERROR

Jadi dua kondisi ini jelas berbeda:

NO CREDENTIAL REQUIRED

vs

CREDENTIAL REQUIRED BUT MISSING
## Phase R5 — Security Boundary Audit

Pastikan secret tidak pernah muncul di:

AIExecutionRequest
Metadata
AIConfiguration
Proto ExecuteRequest
Proto ExecuteResponse
AIError
gRPC status message
logs

Secret hanya boleh berada di:

Encrypted Credential Store
        ↓
Credential Resolver
        ↓
AIExecutionContext.Credential
        ↓
Implementation
        ↓
Provider adapter

Dan setelah execution selesai, tidak boleh ada accidental propagation ke response metadata.

Test yang sekarang sudah mulai membuktikan ini, jadi kita tinggal memperluasnya.

## Phase R6 — Configuration Integrity

Tambahkan invariant tests:

0 config selected
1 config selected
2 configs
Select A
Select B
Delete B
Delete A
Delete inactive config
Select unknown ref
Create duplicate ref

Invariant:

At most one Enabled == true

dan:

active_ref != nil
    ⇒ referenced configuration exists
    ⇒ referenced configuration.Enabled == true
## Phase R7 — Resolver Integrity
Implementation Resolver

Test:

register
resolve
duplicate
unknown
nil
concurrent resolve
Provider Resolver

Sudah cukup bagus, tapi tambahkan:

concurrent Register/Resolve

dengan race detector.

Important invariant

Executor tidak pernah melakukan:

provider switch

atau:

provider type assertion

Provider selection tetap berasal dari configuration/implementation architecture.

## Phase R8 — Mapper Contract

Current mapper sudah bagus.

Tetapkan contract:

Proto
 ↓
Mapper
 ↓
Execution Contract

Mapper tidak boleh:

resolve configuration
resolve credential
resolve provider
execute provider

Test:

Any preserved
metadata preserved
request ID preserved
capability preserved
nil handling
invalid Struct

Sebagian besar sudah ada.

## Phase R9 — Full Internal E2E

Sebelum provider nyata:

gRPC Client
    ↓
gRPC Server
    ↓
Mapper
    ↓
Executor
    ↓
Selection
    ↓
Configuration
    ↓
Credential
    ↓
Implementation Resolver
    ↓
Test Implementation
    ↓
Test Provider

Dengan real:

configuration store
credential store
credential resolver
implementation resolver
provider resolver
gRPC transport

bukan mock seluruh stack.

Target:

PASS = seluruh internal architecture wired
## Phase R10 — Error E2E

Ini penting.

Test:

gRPC Client
 ↓
Server
 ↓
Executor
 ↓
Implementation
 ↓
Provider
 ↓
AUTHENTICATION_ERROR

harus kembali ke client sebagai:

Remote semantic error:
AUTHENTICATION_ERROR

bukan:

UNKNOWN_ERROR

Dan:

api-key=SECRET

tidak boleh muncul di:

status.Message()
AIError.Message
RemoteError.Message
## Phase R11 — Real Provider Readiness Gate

Baru setelah:

go test ./...
go test -race ./...

dan seluruh test matrix hijau:

## R1-R10 = GREEN

kita buka:

13.11 Real Provider Execution