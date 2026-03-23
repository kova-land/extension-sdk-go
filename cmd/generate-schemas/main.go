// Command generate-schemas reflects on the protocol package types and writes
// JSON Schema files to the schemas/ directory.
//
// Flags:
//
//	--check  Validate that committed schemas are up-to-date (exit 1 on drift).
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"

	"github.com/google/jsonschema-go/jsonschema"

	"github.com/kova-land/extension-sdk-go/protocol"
)

func main() {
	check := flag.Bool("check", false, "validate schemas are up-to-date without writing")
	flag.Parse()

	types := []any{
		protocol.Request{},
		protocol.Response{},
		protocol.Notification{},
		protocol.RPCError{},
		protocol.InitializeParams{},
		protocol.InitializeResult{},
		protocol.Registrations{},
		protocol.ToolDef{},
		protocol.HTTPRouteDef{},
		protocol.HookDef{},
		protocol.ServiceDef{},
		protocol.ProviderDef{},
		protocol.TTSProviderDef{},
		protocol.CLIDef{},
		protocol.CallContext{},
		protocol.ToolCallParams{},
		protocol.ToolCallResult{},
		protocol.ChannelMessageParams{},
		protocol.ChannelMessageImage{},
		protocol.ChannelSendParams{},
		protocol.ChannelSendAttachment{},
		protocol.ChannelSendComponent{},
		protocol.ChannelTypingParams{},
		protocol.ChannelReactionAddParams{},
		protocol.ChannelReactionRemoveParams{},
		protocol.PromptOption{},
		protocol.PromptUserParams{},
		protocol.PromptUserResult{},
		protocol.ChannelCapabilities{},
		protocol.ChannelCapabilitiesParams{},
		protocol.ChannelCapabilitiesResult{},
		protocol.HookEventParams{},
		protocol.HookEventResult{},
		protocol.HTTPRequestParams{},
		protocol.HTTPResponseResult{},
		protocol.ServiceStartParams{},
		protocol.ServiceStartResult{},
		protocol.ServiceHealthParams{},
		protocol.ServiceHealthResult{},
		protocol.ServiceStopParams{},
		protocol.ServiceStopResult{},
		protocol.ProviderContentBlock{},
		protocol.ProviderMessage{},
		protocol.ProviderToolDef{},
		protocol.ProviderCreateMessageParams{},
		protocol.ProviderCreateMessageResult{},
		protocol.StreamChunkParams{},
		protocol.StreamEndParams{},
		protocol.StreamErrorParams{},
		protocol.ProviderCapabilitiesParams{},
		protocol.ProviderCapabilitiesResult{},
		protocol.TTSSynthesizeParams{},
		protocol.TTSSynthesizeResult{},
		protocol.TTSVoicesResult{},
		protocol.TTSVoice{},
		protocol.STTProviderDef{},
		protocol.STTTranscribeParams{},
		protocol.STTTranscribeResult{},
		protocol.STTSegment{},
		protocol.STTModelsResult{},
		protocol.STTModel{},
		protocol.LogParams{},
	}

	outDir := "schemas"
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("create schemas dir: %v", err)
	}

	opts := &jsonschema.ForOptions{
		IgnoreInvalidTypes: true,
	}

	var (
		errors []string
		drifts []string
	)

	for _, instance := range types {
		t := reflect.TypeOf(instance)
		name := t.Name()
		filename := name + ".json"

		schema, err := jsonschema.ForType(t, opts)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", filename, err))
			continue
		}

		schema.Title = name

		data, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: marshal: %v", filename, err))
			continue
		}

		data = append(data, '\n')

		outPath := filepath.Join(outDir, filename)

		if *check {
			existing, err := os.ReadFile(outPath)
			if err != nil {
				drifts = append(drifts, fmt.Sprintf("%s: %v", filename, err))
				continue
			}
			if !bytes.Equal(data, existing) {
				drifts = append(drifts, filename)
			}
			continue
		}

		if err := os.WriteFile(outPath, data, 0o644); err != nil {
			errors = append(errors, fmt.Sprintf("%s: write: %v", filename, err))
			continue
		}
		fmt.Printf("wrote %s\n", outPath)
	}

	if len(errors) > 0 {
		for _, e := range errors {
			log.Printf("ERROR: %s", e)
		}
		os.Exit(1)
	}

	if *check {
		if len(drifts) > 0 {
			fmt.Fprintf(os.Stderr, "schemas are out of date — run 'make generate' and commit:\n")
			for _, d := range drifts {
				fmt.Fprintf(os.Stderr, "  %s\n", d)
			}
			os.Exit(1)
		}
		fmt.Printf("All %d schemas are up-to-date\n", len(types))
		return
	}

	fmt.Printf("\nGenerated %d schemas in %s/\n", len(types), outDir)
}
