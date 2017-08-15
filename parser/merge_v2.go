package parser

import (
	"fmt"
	"path"

	"github.com/Sirupsen/logrus"
	"github.com/docker/libcompose/utils"
	"github.com/rancher/rancher-compose-executor/config"
	"github.com/rancher/rancher-compose-executor/lookup"
)

// MergeServicesV2 merges a v2 compose file into an existing set of service configs
func MergeServicesV2(vars map[string]string, resourceLookup lookup.ResourceLookup, file string, datas config.RawServiceMap) (map[string]*config.ServiceConfig, error) {
	if err := validateV2(datas); err != nil {
		return nil, err
	}

	for name, data := range datas {
		var err error
		datas[name], err = parseV2(resourceLookup, vars, file, data, datas)
		if err != nil {
			logrus.Errorf("Failed to parse service %s: %v", name, err)
			return nil, err
		}
	}

	serviceConfigs := make(map[string]*config.ServiceConfig)
	if err := utils.Convert(datas, &serviceConfigs); err != nil {
		return nil, err
	}

	return serviceConfigs, nil
}

func parseV2(resourceLookup lookup.ResourceLookup, vars map[string]string, inFile string, serviceData config.RawService, datas config.RawServiceMap) (config.RawService, error) {
	serviceData, err := readEnvFile(resourceLookup, inFile, serviceData)
	if err != nil {
		return nil, err
	}

	serviceData = resolveContextV2(inFile, serviceData)

	value, ok := serviceData["extends"]
	if !ok {
		return serviceData, nil
	}

	mapValue, ok := value.(map[interface{}]interface{})
	if !ok {
		return serviceData, nil
	}

	if resourceLookup == nil {
		return nil, fmt.Errorf("Can not use extends in file %s no mechanism provided to files", inFile)
	}

	file := asString(mapValue["file"])
	service := asString(mapValue["service"])

	if service == "" {
		return serviceData, nil
	}

	var baseService config.RawService

	if file == "" {
		if serviceData, ok := datas[service]; ok {
			baseService, err = parseV2(resourceLookup, vars, inFile, serviceData, datas)
		} else {
			return nil, fmt.Errorf("Failed to find service %s to extend", service)
		}
	} else {
		bytes, resolved, err := resourceLookup.Lookup(file, inFile)
		if err != nil {
			logrus.Errorf("Failed to lookup file %s: %v", file, err)
			return nil, err
		}

		rawConfig, err := CreateRawConfig(bytes)
		if err != nil {
			return nil, err
		}
		baseRawServices := rawConfig.Services

		if err = InterpolateRawServiceMap(&baseRawServices, vars); err != nil {
			return nil, err
		}

		baseRawServices, err = preProcessServiceMap(baseRawServices)
		if err != nil {
			return nil, err
		}

		if err := validateV2(baseRawServices); err != nil {
			return nil, err
		}

		baseService, ok = baseRawServices[service]
		if !ok {
			return nil, fmt.Errorf("Failed to find service %s in file %s", service, file)
		}

		baseService, err = parseV2(resourceLookup, vars, resolved, baseService, baseRawServices)
	}

	if err != nil {
		return nil, err
	}

	baseService = clone(baseService)

	logrus.Debugf("Merging %#v, %#v", baseService, serviceData)

	for _, k := range noMerge {
		if _, ok := baseService[k]; ok {
			source := file
			if source == "" {
				source = inFile
			}
			return nil, fmt.Errorf("Cannot extend service '%s' in %s: services with '%s' cannot be extended", service, source, k)
		}
	}

	baseService = mergeConfig(baseService, serviceData)

	logrus.Debugf("Merged result %#v", baseService)

	return baseService, nil
}

func resolveContextV2(inFile string, serviceData config.RawService) config.RawService {
	if _, ok := serviceData["build"]; !ok {
		return serviceData
	}
	var build map[interface{}]interface{}
	if buildAsString, ok := serviceData["build"].(string); ok {
		build = map[interface{}]interface{}{
			"context": buildAsString,
		}
	} else {
		build = serviceData["build"].(map[interface{}]interface{})
	}
	context := asString(build["context"])
	if context == "" {
		return serviceData
	}

	if IsValidRemote(context) {
		return serviceData
	}

	current := path.Dir(inFile)

	if context == "." {
		context = current
	} else {
		current = path.Join(current, context)
	}

	build["context"] = current

	return serviceData
}
