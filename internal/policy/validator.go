package policy

import (
	"fmt"
	"strings"
)

// ValidationResult represents policy validation results
type ValidationResult struct {
	Valid  bool
	Errors []string
}

// Validate validates a policy structure
func Validate(policy *Policy) *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Errors: []string{},
	}

	// Check roles
	if len(policy.Roles) == 0 {
		result.addWarning("No roles defined")
	}

	for roleName, role := range policy.Roles {
		if !strings.HasPrefix(roleName, "roles/") {
			result.addError(fmt.Sprintf("Role name must start with 'roles/': %s", roleName))
		}

		if len(role.Permissions) == 0 {
			result.addWarning(fmt.Sprintf("Role %s has no permissions", roleName))
		}

		for _, perm := range role.Permissions {
			if err := validatePermission(perm); err != nil {
				result.addError(fmt.Sprintf("Role %s: %v", roleName, err))
			}
		}
	}

	// Check projects
	if len(policy.Projects) == 0 {
		result.addWarning("No projects defined")
	}

	for projectName, project := range policy.Projects {
		if len(project.Bindings) == 0 {
			result.addWarning(fmt.Sprintf("Project %s has no bindings", projectName))
		}

		for i, binding := range project.Bindings {
			// Check if role exists
			if !strings.HasPrefix(binding.Role, "roles/") {
				result.addError(fmt.Sprintf("Project %s binding %d: role must start with 'roles/'", projectName, i))
			}

			// Check if custom role is defined
			if strings.HasPrefix(binding.Role, "roles/custom.") {
				if _, exists := policy.Roles[binding.Role]; !exists {
					result.addError(fmt.Sprintf("Project %s binding %d: undefined role %s", projectName, i, binding.Role))
				}
			}

			// Check members
			if len(binding.Members) == 0 {
				result.addError(fmt.Sprintf("Project %s binding %d: no members specified", projectName, i))
			}

			for _, member := range binding.Members {
				if err := validatePrincipal(member, policy); err != nil {
					result.addError(fmt.Sprintf("Project %s binding %d: %v", projectName, i, err))
				}
			}

			// Check condition syntax (basic)
			if binding.Condition != nil {
				if binding.Condition.Expression == "" {
					result.addError(fmt.Sprintf("Project %s binding %d: condition has empty expression", projectName, i))
				}
			}
		}
	}

	return result
}

func (r *ValidationResult) addError(msg string) {
	r.Valid = false
	r.Errors = append(r.Errors, msg)
}

func (r *ValidationResult) addWarning(msg string) {
	r.Errors = append(r.Errors, "WARNING: "+msg)
}

func validatePermission(perm string) error {
	parts := strings.Split(perm, ".")
	if len(parts) < 3 {
		return fmt.Errorf("invalid permission format: %s (expected service.resource.verb)", perm)
	}

	service := parts[0]
	if service != "secretmanager" && service != "cloudkms" {
		return fmt.Errorf("unknown service in permission: %s (expected secretmanager or cloudkms)", service)
	}

	return nil
}

func validatePrincipal(principal string, policy *Policy) error {
	if principal == "allUsers" || principal == "allAuthenticatedUsers" {
		return nil
	}

	parts := strings.SplitN(principal, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid principal format: %s (expected type:identifier)", principal)
	}

	principalType := parts[0]
	identifier := parts[1]

	switch principalType {
	case "user", "serviceAccount":
		if !strings.Contains(identifier, "@") {
			return fmt.Errorf("invalid %s: %s (expected email format)", principalType, identifier)
		}
	case "group":
		if _, exists := policy.Groups[identifier]; !exists {
			return fmt.Errorf("undefined group: %s", identifier)
		}
	default:
		return fmt.Errorf("unknown principal type: %s (expected user, serviceAccount, or group)", principalType)
	}

	return nil
}
