package schemas_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"

	"github.com/kova-land/extension-sdk-go/protocol"
)

func allProtocolTypes() map[string]reflect.Type {
	return map[string]reflect.Type{
		"Request":                     reflect.TypeOf(protocol.Request{}),
		"Response":                    reflect.TypeOf(protocol.Response{}),
		"Notification":                reflect.TypeOf(protocol.Notification{}),
		"RPCError":                    reflect.TypeOf(protocol.RPCError{}),
		"InitializeParams":            reflect.TypeOf(protocol.InitializeParams{}),
		"InitializeResult":            reflect.TypeOf(protocol.InitializeResult{}),
		"Registrations":               reflect.TypeOf(protocol.Registrations{}),
		"ToolDef":                     reflect.TypeOf(protocol.ToolDef{}),
		"HTTPRouteDef":                reflect.TypeOf(protocol.HTTPRouteDef{}),
		"HookDef":                     reflect.TypeOf(protocol.HookDef{}),
		"ServiceDef":                  reflect.TypeOf(protocol.ServiceDef{}),
		"ProviderDef":                 reflect.TypeOf(protocol.ProviderDef{}),
		"TTSProviderDef":              reflect.TypeOf(protocol.TTSProviderDef{}),
		"CLIDef":                      reflect.TypeOf(protocol.CLIDef{}),
		"CallContext":                 reflect.TypeOf(protocol.CallContext{}),
		"ToolCallParams":              reflect.TypeOf(protocol.ToolCallParams{}),
		"ToolCallResult":              reflect.TypeOf(protocol.ToolCallResult{}),
		"ChannelMessageParams":        reflect.TypeOf(protocol.ChannelMessageParams{}),
		"ChannelMessageImage":         reflect.TypeOf(protocol.ChannelMessageImage{}),
		"ChannelSendParams":           reflect.TypeOf(protocol.ChannelSendParams{}),
		"ChannelSendAttachment":       reflect.TypeOf(protocol.ChannelSendAttachment{}),
		"ChannelSendComponent":        reflect.TypeOf(protocol.ChannelSendComponent{}),
		"ChannelTypingParams":         reflect.TypeOf(protocol.ChannelTypingParams{}),
		"ChannelReactionAddParams":    reflect.TypeOf(protocol.ChannelReactionAddParams{}),
		"ChannelReactionRemoveParams": reflect.TypeOf(protocol.ChannelReactionRemoveParams{}),
		"PromptOption":                reflect.TypeOf(protocol.PromptOption{}),
		"PromptUserParams":            reflect.TypeOf(protocol.PromptUserParams{}),
		"PromptUserResult":            reflect.TypeOf(protocol.PromptUserResult{}),
		"ChannelCapabilities":         reflect.TypeOf(protocol.ChannelCapabilities{}),
		"ChannelCapabilitiesParams":   reflect.TypeOf(protocol.ChannelCapabilitiesParams{}),
		"ChannelCapabilitiesResult":   reflect.TypeOf(protocol.ChannelCapabilitiesResult{}),
		"HookEventParams":             reflect.TypeOf(protocol.HookEventParams{}),
		"HookEventResult":             reflect.TypeOf(protocol.HookEventResult{}),
		"HTTPRequestParams":           reflect.TypeOf(protocol.HTTPRequestParams{}),
		"HTTPResponseResult":          reflect.TypeOf(protocol.HTTPResponseResult{}),
		"ServiceStartParams":          reflect.TypeOf(protocol.ServiceStartParams{}),
		"ServiceStartResult":          reflect.TypeOf(protocol.ServiceStartResult{}),
		"ServiceHealthParams":         reflect.TypeOf(protocol.ServiceHealthParams{}),
		"ServiceHealthResult":         reflect.TypeOf(protocol.ServiceHealthResult{}),
		"ServiceStopParams":           reflect.TypeOf(protocol.ServiceStopParams{}),
		"ServiceStopResult":           reflect.TypeOf(protocol.ServiceStopResult{}),
		"ProviderContentBlock":        reflect.TypeOf(protocol.ProviderContentBlock{}),
		"ProviderMessage":             reflect.TypeOf(protocol.ProviderMessage{}),
		"ProviderToolDef":             reflect.TypeOf(protocol.ProviderToolDef{}),
		"ProviderCreateMessageParams": reflect.TypeOf(protocol.ProviderCreateMessageParams{}),
		"ProviderCreateMessageResult": reflect.TypeOf(protocol.ProviderCreateMessageResult{}),
		"StreamChunkParams":           reflect.TypeOf(protocol.StreamChunkParams{}),
		"StreamEndParams":             reflect.TypeOf(protocol.StreamEndParams{}),
		"StreamErrorParams":           reflect.TypeOf(protocol.StreamErrorParams{}),
		"ProviderCapabilitiesParams":  reflect.TypeOf(protocol.ProviderCapabilitiesParams{}),
		"ProviderCapabilitiesResult":  reflect.TypeOf(protocol.ProviderCapabilitiesResult{}),
		"TTSSynthesizeParams":         reflect.TypeOf(protocol.TTSSynthesizeParams{}),
		"TTSSynthesizeResult":         reflect.TypeOf(protocol.TTSSynthesizeResult{}),
		"TTSVoicesResult":             reflect.TypeOf(protocol.TTSVoicesResult{}),
		"TTSVoice":                    reflect.TypeOf(protocol.TTSVoice{}),
		"LogParams":                   reflect.TypeOf(protocol.LogParams{}),
	}
}

func TestSchemaFilesExist(t *testing.T) {
	for name := range allProtocolTypes() {
		filename := name + ".json"
		path := filepath.Join(".", filename)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("missing schema file for %s: %s", name, path)
		}
	}
}

func TestSchemaFilesAreValidJSON(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read schemas dir: %v", err)
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			data, err := os.ReadFile(entry.Name())
			if err != nil {
				t.Fatalf("read: %v", err)
			}

			var schema map[string]any
			if err := json.Unmarshal(data, &schema); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}

			if _, ok := schema["type"]; !ok {
				t.Error("schema missing 'type' field")
			}

			title, ok := schema["title"]
			if !ok {
				t.Error("schema missing 'title' field")
			}

			expected := strings.TrimSuffix(entry.Name(), ".json")
			if title != expected {
				t.Errorf("title mismatch: got %q, want %q", title, expected)
			}
		})
	}
}

func TestSchemasMatchTypes(t *testing.T) {
	opts := &jsonschema.ForOptions{
		IgnoreInvalidTypes: true,
	}

	for name, typ := range allProtocolTypes() {
		t.Run(name, func(t *testing.T) {
			freshSchema, err := jsonschema.ForType(typ, opts)
			if err != nil {
				t.Fatalf("ForType(%s): %v", name, err)
			}

			freshData, err := json.Marshal(freshSchema)
			if err != nil {
				t.Fatalf("marshal fresh schema: %v", err)
			}

			filename := filepath.Join(".", name+".json")
			fileData, err := os.ReadFile(filename)
			if err != nil {
				t.Fatalf("read schema file: %v", err)
			}

			var freshMap, fileMap map[string]any
			if err := json.Unmarshal(freshData, &freshMap); err != nil {
				t.Fatalf("unmarshal fresh: %v", err)
			}
			if err := json.Unmarshal(fileData, &fileMap); err != nil {
				t.Fatalf("unmarshal file: %v", err)
			}

			freshProps := extractPropertyNames(freshMap)
			fileProps := extractPropertyNames(fileMap)

			for _, p := range freshProps {
				if !contains(fileProps, p) {
					t.Errorf("property %q in Go type but missing from schema file", p)
				}
			}
			for _, p := range fileProps {
				if !contains(freshProps, p) {
					t.Errorf("property %q in schema file but missing from Go type", p)
				}
			}
		})
	}
}

func TestNoExtraSchemaFiles(t *testing.T) {
	types := allProtocolTypes()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read schemas dir: %v", err)
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".json")
		if _, ok := types[name]; !ok {
			t.Errorf("schema file %s has no corresponding protocol type", entry.Name())
		}
	}
}

func extractPropertyNames(schema map[string]any) []string {
	props, ok := schema["properties"]
	if !ok {
		return nil
	}
	propsMap, ok := props.(map[string]any)
	if !ok {
		return nil
	}
	names := make([]string, 0, len(propsMap))
	for k := range propsMap {
		names = append(names, k)
	}
	return names
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
