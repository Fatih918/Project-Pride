package configuration

import "testing"

func TestAIConfigurationValidate(t *testing.T) {
	tests := []struct {
		name          string
		configuration AIConfiguration
		wantErr       bool
	}{
		{
			name: "valid configuration",
			configuration: AIConfiguration{
				Ref:               "config_test_openai",
				Name:              "test-openai",
				Type:              "openai",
				ImplementationRef: "openai.responses",
			},
			wantErr: false,
		},
		{
			name: "missing reference",
			configuration: AIConfiguration{
				Name:              "test-openai",
				Type:              "openai",
				ImplementationRef: "openai.responses",
			},
			wantErr: true,
		},
		{
			name: "missing name",
			configuration: AIConfiguration{
				Ref:               "config_test_openai",
				Type:              "openai",
				ImplementationRef: "openai.responses",
			},
			wantErr: true,
		},
		{
			name: "missing type",
			configuration: AIConfiguration{
				Ref:               "config_test_openai",
				Name:              "test-openai",
				ImplementationRef: "openai.responses",
			},
			wantErr: true,
		},
		{
			name: "missing implementation reference",
			configuration: AIConfiguration{
				Ref:  "config_test_openai",
				Name: "test-openai",
				Type: "openai",
			},
			wantErr: true,
		},
		{
			name: "credential reference may be empty",
			configuration: AIConfiguration{
				Ref:               "config_test",
				Name:              "test",
				Type:              "test",
				ImplementationRef: "test.implementation",
				CredentialRef:     "",
			},
			wantErr: false,
		},
		{
			name: "disabled configuration is valid",
			configuration: AIConfiguration{
				Ref:               "config_test",
				Name:              "test",
				Type:              "test",
				ImplementationRef: "test.implementation",
				Enabled:           false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.configuration.Validate()

			if tt.wantErr && err == nil {
				t.Fatal("Validate() expected an error")
			}

			if !tt.wantErr && err != nil {
				t.Fatalf(
					"Validate() returned unexpected error: %v",
					err,
				)
			}
		})
	}
}
