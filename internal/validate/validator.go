package validate

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ValidationResult struct {
	Valid    bool     `json:"valid"`
	File     string   `json:"file"`
	Type     string   `json:"type"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

type RecipeSpec struct {
	APIVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   map[string]string `yaml:"metadata"`
	Runtime    RuntimeSpec       `yaml:"runtime"`
}

type RuntimeSpec struct {
	Image     string            `yaml:"image"`
	Command   []string          `yaml:"command,omitempty"`
	Env       map[string]string `yaml:"env,omitempty"`
	Resources ResourceSpec      `yaml:"resources,omitempty"`
}

type ResourceSpec struct {
	CPU    string `yaml:"cpu,omitempty"`
	Memory string `yaml:"memory,omitempty"`
}

type ContextSpec struct {
	APIVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   map[string]string `yaml:"metadata"`
	Inputs     []InputSpec       `yaml:"inputs,omitempty"`
	Outputs    []OutputSpec      `yaml:"outputs,omitempty"`
}

type InputSpec struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Description string `yaml:"description,omitempty"`
	Required    bool   `yaml:"required,omitempty"`
}

type OutputSpec struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Description string `yaml:"description,omitempty"`
}

type SimulationSpec struct {
	APIVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   map[string]string `yaml:"metadata"`
	Spec       SimSpec           `yaml:"spec"`
}

type SimSpec struct {
	RecipeRef  string            `yaml:"recipeRef,omitempty"`
	ContextRef string            `yaml:"contextRef,omitempty"`
	Timeout    string            `yaml:"timeout,omitempty"`
	Parameters map[string]string `yaml:"parameters,omitempty"`
}

func ValidateRecipe(file string) (*ValidationResult, error) {
	return validateFile(file, "recipe", validateRecipeContent)
}

func ValidateContext(file string) (*ValidationResult, error) {
	return validateFile(file, "context", validateContextContent)
}

func ValidateSimulation(file string) (*ValidationResult, error) {
	return validateFile(file, "simulation", validateSimulationContent)
}

func validateFile(file, fileType string, validator func([]byte) ([]string, []string)) (*ValidationResult, error) {
	result := &ValidationResult{
		File:  file,
		Type:  fileType,
		Valid: true,
	}

	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", file, err)
	}

	errs, warnings := validator(data)
	result.Errors = errs
	result.Warnings = warnings

	if len(errs) > 0 {
		result.Valid = false
	}

	return result, nil
}

func validateRecipeContent(data []byte) (errors, warnings []string) {
	var spec RecipeSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return []string{fmt.Sprintf("invalid YAML: %v", err)}, nil
	}

	if spec.APIVersion == "" {
		errors = append(errors, "missing required field: apiVersion")
	}
	if spec.Kind == "" {
		errors = append(errors, "missing required field: kind")
	} else if spec.Kind != "Recipe" {
		warnings = append(warnings, fmt.Sprintf("unexpected kind %q, expected \"Recipe\"", spec.Kind))
	}
	if spec.Runtime.Image == "" {
		errors = append(errors, "missing required field: runtime.image")
	}

	return errors, warnings
}

func validateContextContent(data []byte) (errors, warnings []string) {
	var spec ContextSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return []string{fmt.Sprintf("invalid YAML: %v", err)}, nil
	}

	if spec.APIVersion == "" {
		errors = append(errors, "missing required field: apiVersion")
	}
	if spec.Kind == "" {
		errors = append(errors, "missing required field: kind")
	} else if spec.Kind != "Context" {
		warnings = append(warnings, fmt.Sprintf("unexpected kind %q, expected \"Context\"", spec.Kind))
	}

	for i, input := range spec.Inputs {
		if input.Name == "" {
			errors = append(errors, fmt.Sprintf("inputs[%d]: missing required field: name", i))
		}
		if input.Type == "" {
			errors = append(errors, fmt.Sprintf("inputs[%d]: missing required field: type", i))
		}
	}

	return errors, warnings
}

func validateSimulationContent(data []byte) (errors, warnings []string) {
	var spec SimulationSpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return []string{fmt.Sprintf("invalid YAML: %v", err)}, nil
	}

	if spec.APIVersion == "" {
		errors = append(errors, "missing required field: apiVersion")
	}
	if spec.Kind == "" {
		errors = append(errors, "missing required field: kind")
	} else if spec.Kind != "Simulation" {
		warnings = append(warnings, fmt.Sprintf("unexpected kind %q, expected \"Simulation\"", spec.Kind))
	}
	if spec.Spec.RecipeRef == "" {
		warnings = append(warnings, "spec.recipeRef is empty — simulation may not execute correctly")
	}

	return errors, warnings
}
