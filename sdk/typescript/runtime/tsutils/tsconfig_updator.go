package tsutils

import (
	"fmt"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	daggerLibPathAlias       = "@github.com/G-Research/dagger"
	daggerTelemetryPathAlias = "@github.com/G-Research/dagger/telemetry"

	daggerLibPath          = "./sdk/index.ts"
	daggerTelemetryLibPath = "./sdk/telemetry.ts"
)

// Generate a default tsconfig.json file
func DefaultTSConfig() string {
	return DefaultTSConfigJSON
}

// Update the tsconfig.json file for a module
func UpdateTSConfigForModule(tsConfig string) (string, error) {
	tsConfig = removeJSONComments(tsConfig)

	// Add path."@github.com/G-Research/dagger"=["./sdk/index.ts"]
	tsConfig, err := sjson.Set(tsConfig,
		"compilerOptions.paths."+gjson.Escape(daggerLibPathAlias),
		[]string{daggerLibPath},
	)
	if err != nil {
		return "", fmt.Errorf("failed to update tsconfig paths: %w", err)
	}

	// Add path."@github.com/G-Research/dagger/telemetry"=["./sdk/telemetry.ts"]
	tsConfig, err = sjson.Set(tsConfig,
		"compilerOptions.paths."+gjson.Escape(daggerTelemetryPathAlias),
		[]string{daggerTelemetryLibPath},
	)
	if err != nil {
		return "", fmt.Errorf("failed to update tsconfig paths %s: %w", daggerTelemetryPathAlias, err)
	}

	// Add compilerOptions.experimentalDecorators=true
	tsConfig, err = sjson.Set(tsConfig, "compilerOptions.experimentalDecorators", true)
	if err != nil {
		return "", fmt.Errorf("failed to update tsconfig experimentalDecorators: %w", err)
	}

	return tsConfig, nil
}

func UpdateTSConfigForClient(tsConfig string, isRemote bool) (string, error) {
	tsConfig = removeJSONComments(tsConfig)

	// If the dagger library is remote, we don't need to override @github.com/G-Research/dagger
	// and @github.com/G-Research/dagger/telemetry
	if isRemote {
		return tsConfig, nil
	}

	// Add path."@github.com/G-Research/dagger/telemetry"=["./sdk/telemetry.ts"]
	tsConfig, err := sjson.Set(tsConfig,
		"compilerOptions.paths."+gjson.Escape(daggerTelemetryPathAlias),
		[]string{daggerTelemetryLibPath},
	)
	if err != nil {
		return "", fmt.Errorf("failed to update tsconfig paths %s: %w", daggerTelemetryPathAlias, err)
	}

	// Add path."@github.com/G-Research/dagger"=["./sdk/index.ts"]
	tsConfig, err = sjson.Set(tsConfig,
		"compilerOptions.paths."+gjson.Escape(daggerLibPathAlias),
		[]string{daggerLibPath},
	)
	if err != nil {
		return "", fmt.Errorf("failed to update tsconfig paths: %w", err)
	}

	return tsConfig, nil
}
