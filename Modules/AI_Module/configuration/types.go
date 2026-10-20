package configuration

import "fmt"

type AIConfiguration struct {
	Ref                  string
	Name                 string
	Type                 string
	ImplementationRef    string
	ConnectionParameters map[string]any
	CredentialRef        string
	Enabled              bool
}

func (c AIConfiguration) Validate() error {
	if c.Ref == "" {
		return fmt.Errorf(
			"AI configuration reference must not be empty",
		)
	}

	if c.Name == "" {
		return fmt.Errorf(
			"AI configuration name must not be empty",
		)
	}

	if c.Type == "" {
		return fmt.Errorf(
			"AI configuration type must not be empty",
		)
	}

	if c.ImplementationRef == "" {
		return fmt.Errorf(
			"AI configuration implementation reference must not be empty",
		)
	}

	return nil
}
