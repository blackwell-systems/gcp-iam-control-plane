package cli

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/blackwell-systems/gcp-emulator-control-plane/internal/config"
	"github.com/blackwell-systems/gcp-emulator-control-plane/internal/policy"
)

var policyCmd = &cobra.Command{
	Use:   "policy",
	Short: "Policy management",
	Long:  `Validate, initialize, and manage policy.yaml files.`,
}

var policyValidateCmd = &cobra.Command{
	Use:   "validate [file]",
	Short: "Validate policy.yaml syntax",
	Long: `Validate policy file syntax and structure.

Without arguments, validates ./policy.yaml
Specify a file path to validate a different file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		policyFile := cfg.PolicyFile
		if len(args) > 0 {
			policyFile = args[0]
		}

		color.Cyan("Validating %s...", policyFile)

		// Load policy
		pol, err := policy.Load(policyFile)
		if err != nil {
			color.Red("✗ Failed to load policy: %v", err)
			return err
		}

		// Validate
		result := policy.Validate(pol)

		if result.Valid {
			color.Green("✓ Policy is valid")
			fmt.Printf("\n%d roles defined\n", len(pol.Roles))
			fmt.Printf("%d groups defined\n", len(pol.Groups))
			fmt.Printf("%d projects configured\n", len(pol.Projects))

			// Show warnings if any
			for _, err := range result.Errors {
				if len(err) > 0 {
					color.Yellow("  %s", err)
				}
			}

			return nil
		}

		color.Red("✗ Validation failed")
		fmt.Println("\nErrors:")
		for _, err := range result.Errors {
			color.Red("  %s", err)
		}

		return fmt.Errorf("policy validation failed")
	},
}

var policyInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new policy file",
	Long: `Create a new policy.yaml file from a template.

Templates:
  basic    - Simple developer + CI roles
  advanced - Multiple roles with conditions
  ci       - CI-focused configuration`,
	RunE: func(cmd *cobra.Command, args []string) error {
		template, _ := cmd.Flags().GetString("template")
		force, _ := cmd.Flags().GetBool("force")
		output, _ := cmd.Flags().GetString("output")

		// Check if file exists
		if !force {
			if _, err := os.Stat(output); err == nil {
				return fmt.Errorf("file %s already exists (use --force to overwrite)", output)
			}
		}

		color.Cyan("Creating policy file: %s", output)
		color.Cyan("Template: %s", template)

		// Create policy from template
		pol := createPolicyFromTemplate(template)

		// Save to file
		if err := policy.Save(pol, output); err != nil {
			color.Red("✗ Failed to save policy: %v", err)
			return err
		}

		color.Green("✓ Policy file created successfully")
		fmt.Println("\nEdit the file to customize for your project:")
		fmt.Printf("  vim %s\n", output)
		fmt.Println("\nThen start the stack:")
		fmt.Println("  gcp-emulator start")

		return nil
	},
}

func createPolicyFromTemplate(template string) *policy.Policy {
	switch template {
	case "advanced":
		return &policy.Policy{
			Roles: map[string]policy.Role{
				"roles/custom.developer": {
					Permissions: []string{
						"secretmanager.secrets.create",
						"secretmanager.secrets.get",
						"secretmanager.secrets.update",
						"secretmanager.versions.add",
						"secretmanager.versions.access",
						"cloudkms.keyRings.create",
						"cloudkms.cryptoKeys.create",
						"cloudkms.cryptoKeys.encrypt",
						"cloudkms.cryptoKeys.decrypt",
					},
				},
				"roles/custom.ciRunner": {
					Permissions: []string{
						"secretmanager.secrets.get",
						"secretmanager.versions.access",
						"cloudkms.cryptoKeys.encrypt",
					},
				},
				"roles/custom.readonly": {
					Permissions: []string{
						"secretmanager.secrets.get",
						"cloudkms.keyRings.get",
						"cloudkms.cryptoKeys.get",
					},
				},
			},
			Groups: map[string]policy.Group{
				"developers": {
					Members: []string{
						"user:alice@example.com",
						"user:bob@example.com",
					},
				},
				"operations": {
					Members: []string{
						"user:ops@example.com",
					},
				},
			},
			Projects: map[string]policy.Project{
				"test-project": {
					Bindings: []policy.Binding{
						{
							Role:    "roles/custom.developer",
							Members: []string{"group:developers"},
						},
						{
							Role:    "roles/custom.ciRunner",
							Members: []string{"serviceAccount:ci@test-project.iam.gserviceaccount.com"},
							Condition: &policy.Condition{
								Expression: `resource.name.startsWith("projects/test-project/secrets/prod-")`,
								Title:      "CI limited to production secrets",
							},
						},
						{
							Role:    "roles/custom.readonly",
							Members: []string{"group:operations"},
						},
					},
				},
			},
		}
	case "ci":
		return &policy.Policy{
			Roles: map[string]policy.Role{
				"roles/custom.ciRunner": {
					Permissions: []string{
						"secretmanager.secrets.get",
						"secretmanager.versions.access",
					},
				},
			},
			Groups: map[string]policy.Group{
				"ci-accounts": {
					Members: []string{
						"serviceAccount:ci@test-project.iam.gserviceaccount.com",
						"serviceAccount:github-actions@test-project.iam.gserviceaccount.com",
					},
				},
			},
			Projects: map[string]policy.Project{
				"test-project": {
					Bindings: []policy.Binding{
						{
							Role:    "roles/custom.ciRunner",
							Members: []string{"group:ci-accounts"},
						},
					},
				},
			},
		}
	default: // "basic"
		return &policy.Policy{
			Roles: map[string]policy.Role{
				"roles/custom.developer": {
					Permissions: []string{
						"secretmanager.secrets.create",
						"secretmanager.secrets.get",
						"secretmanager.versions.add",
						"secretmanager.versions.access",
						"cloudkms.cryptoKeys.encrypt",
						"cloudkms.cryptoKeys.decrypt",
					},
				},
			},
			Groups: map[string]policy.Group{
				"developers": {
					Members: []string{
						"user:alice@example.com",
					},
				},
			},
			Projects: map[string]policy.Project{
				"test-project": {
					Bindings: []policy.Binding{
						{
							Role:    "roles/custom.developer",
							Members: []string{"group:developers"},
						},
					},
				},
			},
		}
	}
}

func init() {
	policyCmd.AddCommand(policyValidateCmd)
	policyCmd.AddCommand(policyInitCmd)

	policyInitCmd.Flags().String("template", "basic", "Template to use (basic|advanced|ci)")
	policyInitCmd.Flags().BoolP("force", "f", false, "Overwrite existing policy.yaml")
	policyInitCmd.Flags().String("output", "policy.yaml", "Output file path")
}
