package e2e


	import (
    "bufio"
    "context"
    "io"
    "os"
    "os/exec"
    "path/filepath"
    "testing"
    "time"

    aiv1 "pride-ai-sdk/proto"

    "google.golang.org/grpc"
    "google.golang.org/grpc/connectivity"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/protobuf/proto"
    "google.golang.org/protobuf/types/known/anypb"

)

const (
	sdkAddress    = "127.0.0.1:50051"
	moduleAddress = "127.0.0.1:50052"

	serverReadyTimeout = 15 * time.Second
	testTimeout        = 30 * time.Second
)

func TestSDKToAIModuleEndToEnd(t *testing.T) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		testTimeout,
	)
	defer cancel()

	// ------------------------------------------------------------
	// 1. Locate AI Module
	// ------------------------------------------------------------

	moduleDir := filepath.Join("..", "Modules", "AI_Module")


	// ------------------------------------------------------------
	// 2. Locate AI SDK
	// ------------------------------------------------------------

	sdkDir := filepath.Join("..", "sdks", "ai")

	// ------------------------------------------------------------
	// 3. Start AI Module
	// ------------------------------------------------------------

	moduleCmd := exec.CommandContext(
		ctx,
		"go",
		"run",
		"./pride-ai-module/cmd/ai-module",
	)

	moduleCmd.Dir = moduleDir

	moduleCmd.Env = append(
		os.Environ(),
		"PRIDE_AI_MODULE_ADDR="+moduleAddress,
	)

	moduleStdout, err := moduleCmd.StdoutPipe()
	if err != nil {
		t.Fatalf(
			"failed to create AI Module stdout pipe: %v",
			err,
		)
	}

	moduleStderr, err := moduleCmd.StderrPipe()
	if err != nil {
		t.Fatalf(
			"failed to create AI Module stderr pipe: %v",
			err,
		)
	}

	logProcessOutput(
		t,
		"AI-MODULE",
		moduleStdout,
		moduleStderr,
	)

	if err := moduleCmd.Start(); err != nil {
		t.Fatalf(
			"failed to start AI Module: %v",
			err,
		)
	}

	t.Cleanup(func() {
		stopProcess(moduleCmd)
	})

	// ------------------------------------------------------------
	// 4. Wait for Module
	// ------------------------------------------------------------

	waitForGRPCServer(
		t,
		ctx,
		moduleAddress,
		serverReadyTimeout,
	)

	// ------------------------------------------------------------
	// 5. Start AI SDK
	// ------------------------------------------------------------

	sdkCmd := exec.CommandContext(
		ctx,
		"go",
		"run",
		"./server/cmd/ai-sdk-server",
	)

	sdkCmd.Dir = sdkDir

	sdkCmd.Env = append(
		os.Environ(),
		"PRIDE_AI_MODULE_ADDR="+moduleAddress,
		"PRIDE_AI_SDK_ADDR="+sdkAddress,
	)

	sdkStdout, err := sdkCmd.StdoutPipe()
	if err != nil {
		t.Fatalf(
			"failed to create AI SDK stdout pipe: %v",
			err,
		)
	}

	sdkStderr, err := sdkCmd.StderrPipe()
	if err != nil {
		t.Fatalf(
			"failed to create AI SDK stderr pipe: %v",
			err,
		)
	}

	logProcessOutput(
		t,
		"AI-SDK",
		sdkStdout,
		sdkStderr,
	)

	if err := sdkCmd.Start(); err != nil {
		t.Fatalf(
			"failed to start AI SDK: %v",
			err,
		)
	}

	t.Cleanup(func() {
		stopProcess(sdkCmd)
	})

	// ------------------------------------------------------------
	// 6. Wait for SDK
	// ------------------------------------------------------------

	waitForGRPCServer(
		t,
		ctx,
		sdkAddress,
		serverReadyTimeout,
	)

	// ------------------------------------------------------------
	// 7. Connect external client → SDK
	// ------------------------------------------------------------

	conn, err := grpc.NewClient(
		sdkAddress,
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
	)
	if err != nil {
		t.Fatalf(
			"failed to create SDK gRPC client: %v",
			err,
		)
	}

	t.Cleanup(func() {
		_ = conn.Close()
	})

	client := aiv1.NewAIServiceClient(conn)

	// ------------------------------------------------------------
	// 8. Build request
	// ------------------------------------------------------------

	input := &anypb.Any{
		TypeUrl: "pride.test/Input",
		Value:   []byte{1, 2, 3, 4},
	}

	request := &aiv1.ExecuteRequest{
		RequestId:  "step-12-10-e2e-001",
		Capability: "text.generate",
		Input:      input,
	}

	// ------------------------------------------------------------
	// 9. Execute
	//
	// External Test Client
	//       ↓
	//       ↓ gRPC
	//       ↓
	// SDK Server
	//       ↓
	// SDK Executor
	//       ↓
	// SDK GRPCAIClient
	//       ↓
	//       ↓ gRPC
	//       ↓
	// AI Module Server
	//       ↓
	// Module Executor
	//       ↓
	// Implementation
	// ------------------------------------------------------------

	response, err := client.Execute(
		ctx,
		request,
	)
	if err != nil {
		t.Fatalf(
			"SDK Execute() failed: %v",
			err,
		)
	}

	// ------------------------------------------------------------
	// 10. Validate response
	// ------------------------------------------------------------

	if response == nil {
		t.Fatal("SDK returned nil response")
	}

	if response.GetRequestId() != request.GetRequestId() {
		t.Fatalf(
			"request ID mismatch: got %q, want %q",
			response.GetRequestId(),
			request.GetRequestId(),
		)
	}

	if response.GetCapability() != request.GetCapability() {
		t.Fatalf(
			"capability mismatch: got %q, want %q",
			response.GetCapability(),
			request.GetCapability(),
		)
	}

	if response.GetOutput() == nil {
		t.Fatal("response output must not be nil")
	}

	// ------------------------------------------------------------
	// 11. Validate Module metadata
	// ------------------------------------------------------------

	if response.GetMetadata() == nil {
		t.Fatal("response metadata must not be nil")
	}

	sourceField, ok := response.GetMetadata().
		GetFields()["source"]

	if !ok {
		t.Fatal(
			"response metadata does not contain source",
		)
	}

	source := sourceField.GetStringValue()

	if source != "ai-module" {
		t.Fatalf(
			"unexpected response metadata source: got %q, want %q",
			source,
			"ai-module",
		)
	}

	// ------------------------------------------------------------
	// 12. Validate Any payload
	// ------------------------------------------------------------

	outputMessage := &aiv1.ExecuteResponse{}

	if err := response.GetOutput().UnmarshalTo(
		outputMessage,
	); err != nil {
		t.Fatalf(
			"failed to unmarshal response output: %v",
			err,
		)
	}

	if outputMessage.GetRequestId() != request.GetRequestId() {
		t.Fatalf(
			"output request ID mismatch: got %q, want %q",
			outputMessage.GetRequestId(),
			request.GetRequestId(),
		)
	}

	if outputMessage.GetCapability() != request.GetCapability() {
		t.Fatalf(
			"output capability mismatch: got %q, want %q",
			outputMessage.GetCapability(),
			request.GetCapability(),
		)
	}

	if !proto.Equal(
		outputMessage.GetOutput(),
		input,
	) {
		t.Fatal(
			"Any payload was modified during SDK ↔ Module execution",
		)
	}

	// ------------------------------------------------------------
	// 13. Verify processes remain alive
	// ------------------------------------------------------------

	assertProcessAlive(
		t,
		moduleCmd,
		"AI Module",
	)

	assertProcessAlive(
		t,
		sdkCmd,
		"AI SDK",
	)
}

// ============================================================
// Project discovery
// ============================================================

func findProjectDir(
	t *testing.T,
	envName string,
	requiredPath string,
) string {
	t.Helper()

	if configured := os.Getenv(envName); configured != "" {
		if !isRequiredPathPresent(
			configured,
			requiredPath,
		) {
			t.Fatalf(
				"%s=%q does not contain %s",
				envName,
				configured,
				requiredPath,
			)
		}

		return configured
	}

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf(
			"failed to determine working directory: %v",
			err,
		)
	}

	dir := workingDir

	for {
		if isRequiredPathPresent(
			dir,
			requiredPath,
		) {
			return dir
		}

		parent := filepath.Dir(dir)

		if parent == dir {
			break
		}

		dir = parent
	}

	t.Fatalf(
		"could not locate project containing %s; set %s",
		requiredPath,
		envName,
	)

	return ""
}

func isRequiredPathPresent(
	root string,
	requiredPath string,
) bool {
	path := filepath.Join(
		root,
		filepath.FromSlash(requiredPath),
	)

	info, err := os.Stat(path)

	if err != nil {
		return false
	}

	return info != nil
}

// ============================================================
// gRPC readiness
// ============================================================

func waitForGRPCServer(
	t *testing.T,
	parentCtx context.Context,
	address string,
	timeout time.Duration,
) {
	t.Helper()

	ctx, cancel := context.WithTimeout(parentCtx, timeout)
	defer cancel()

	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		),
	)
	if err != nil {
		t.Fatalf(
			"failed to create gRPC connection to %s: %v",
			address,
			err,
		)
	}
	defer conn.Close()

	conn.Connect()

	for {
		state := conn.GetState()

		if state == connectivity.Ready {
			return
		}

		if state == connectivity.TransientFailure {
			conn.Connect()
		}

		if !conn.WaitForStateChange(ctx, state) {
			t.Fatalf(
				"timed out waiting for gRPC server %s: %v",
				address,
				ctx.Err(),
			)
		}
	}
}

func waitForReady(
	ctx context.Context,
	conn *grpc.ClientConn,
) error {
	for {
		state := conn.GetState()

		if state == connectivity.Ready {
			return nil
		}

		if !conn.WaitForStateChange(
			ctx,
			state,
		) {
			return ctx.Err()
		}
	}
}

// ============================================================
// Process lifecycle
// ============================================================

func stopProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}

	if cmd.ProcessState != nil &&
		cmd.ProcessState.Exited() {
		return
	}

	_ = cmd.Process.Signal(
		os.Interrupt,
	)

	done := make(chan struct{})

	go func() {
		_ = cmd.Wait()
		close(done)
	}()

	select {
	case <-done:
		return

	case <-time.After(3 * time.Second):
		_ = cmd.Process.Kill()

		select {
		case <-done:
		case <-time.After(time.Second):
		}
	}
}

func assertProcessAlive(
	t *testing.T,
	cmd *exec.Cmd,
	name string,
) {
	t.Helper()

	if cmd == nil || cmd.Process == nil {
		t.Fatalf(
			"%s process was not started",
			name,
		)
	}

	if cmd.ProcessState != nil &&
		cmd.ProcessState.Exited() {
		t.Fatalf(
			"%s process exited unexpectedly: %v",
			name,
			cmd.ProcessState,
		)
	}
}

// ============================================================
// Process logging
// ============================================================

func logProcessOutput(
	t *testing.T,
	name string,
	stdout io.Reader,
	stderr io.Reader,
) {
	t.Helper()

	go func() {
		scanner := bufio.NewScanner(stdout)

		for scanner.Scan() {
			t.Logf(
				"[%s] %s",
				name,
				scanner.Text(),
			)
		}
		if err := scanner.Err(); err != nil {
			t.Logf("[%s] scanner error: %v", name, err)
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)

		for scanner.Scan() {
			t.Logf(
				"[%s STDERR] %s",
				name,
				scanner.Text(),
			)
		}
		if err := scanner.Err(); err != nil {
			t.Logf("[%s STDERR] scanner error: %v", name, err)
		}
	}()
}