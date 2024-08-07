package pkg

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/eunanhardy/nori/internal/futils"
	"github.com/eunanhardy/nori/internal/hcl"
	"github.com/eunanhardy/nori/internal/spec"
)

func PackageModuleV2(tag *spec.Tag, packagePathFlag string) error {
	if packagePathFlag == "" {
		fmt.Println("Path is required")
		os.Exit(1)
	}

	moduleCompressedData, err := futils.CompressModule(packagePathFlag, tag.Name); if err != nil {
		return fmt.Errorf("error compressing module: %s", err)
	}

	mediaDigest, err := futils.WriteBlob(moduleCompressedData, tag, spec.MEDIA_TYPE_MODULE_PRIMARY)
	if err != nil {
		return fmt.Errorf("error packaging blob: %s", err)
	}

	moduleData, err := hcl.ParseModuleConfig(packagePathFlag); if err != nil {
		return fmt.Errorf("error parsing module config: %s", err)
	}

	configDigest, err := generateConfig(moduleData, tag); if err != nil {
		return fmt.Errorf("error generating config: %s", err)
	}

	manifestDigest, err := generateManifest(*mediaDigest, *configDigest, tag); if err != nil {
		return fmt.Errorf("error generating manifest: %s", err)
	}

	err = futils.CreateOrUpdateIndex(tag,manifestDigest.Digest); if err != nil {
		return fmt.Errorf("error creating or updating index: %s", err)
	}

	fmt.Println("Module packaged with tag: ", tag.String())


	return nil
}

func generateManifest(layersDigest, config spec.Digest, tag *spec.Tag) (*spec.Digest,error) {

	var manifest = spec.Manifest{
		Schema:    2,
		MediaType: spec.MEDIA_TYPE_MANIFEST,
		Config:   config,
		Layers: []spec.Digest{
			layersDigest,
		},
		Annotations: map[string]string{
			spec.ANNO_IMAGE_REF_NAME: tag.String(),
		},
	}

	jsonBytes, err := json.Marshal(manifest); if err != nil {
		fmt.Println("Error marshalling manifest: ", err)
		return nil,err
	}

	digest, err := futils.WriteBlob(jsonBytes, tag, spec.MEDIA_TYPE_MANIFEST); if err != nil {
		fmt.Println("Error writing manifest: ", err)
		return nil,err
	}

	return digest,nil
}

func validatePackageFlags(packageFlag string, pathFlag string) {
	if packageFlag == "" {
		fmt.Println("Tag is required")
		os.Exit(1)
	}
	if pathFlag == "" {
		fmt.Println("Path is required")
		os.Exit(1)
	}
}

func generateConfig(data *hcl.ModuleConfig, tag *spec.Tag) (*spec.Digest,error){
	var inputs = make(map[string]spec.ModuleInputs)
	var outputs = make(map[string]spec.ModuleOutputs)
	for _, value := range data.Inputs {
		var input = spec.ModuleInputs{
			Description: value.Description,
			Default: value.Default,
		}
		inputs[value.Name] = input
	}

	for _, value := range data.Outputs {
		var output = spec.ModuleOutputs{
			Description: value.Description,
			Sensitive: value.Sensitive,
		}
		outputs[value.Name] = output
	}

	config := spec.Config{
		SchemaVersion: 1,
		MediaType: spec.MEDIA_TYPE_CONFIG,
		Name: tag.Name,
		Version: tag.Version,
		Remote: tag.Host,
		Inputs: inputs,
		Outputs: outputs,
	}

	jsonBytes, err := json.Marshal(config); if err != nil {
		fmt.Println("Error marshalling config: ", err)
		return nil,err
	}

	digest,err := futils.WriteBlob(jsonBytes, tag,spec.MEDIA_TYPE_CONFIG); if err != nil {
		fmt.Println("Error compressing empty json: ", err)
		return nil,err
	}
	
	return digest,nil
}