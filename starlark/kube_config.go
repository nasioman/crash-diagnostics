// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package starlark

import (
	"fmt"

	"github.com/pkg/errors"
	"go.starlark.net/starlark"
	"go.starlark.net/starlarkstruct"
)

// KubeConfigFn is built-in starlark function that wraps the kwargs into a dictionary value.
// The result is also added to the thread for other built-in to access.
// Starlark: kube_config(path=kubecf/path)
func KubeConfigFn(thread *starlark.Thread, _ *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path string
	var provider *starlarkstruct.Struct

	if err := starlark.UnpackArgs(
		identifiers.crashdCfg, args, kwargs,
		"path?", &path,
		"capi_provider?", &provider,
	); err != nil {
		return starlark.None, fmt.Errorf("%s: %s", identifiers.kubeCfg, err)
	}

	// check if only one of the two options are present
	if (len(path) == 0 && provider == nil) || (len(path) != 0 && provider != nil) {
		return starlark.None, errors.New("need either path or capi_provider")
	}

	if len(path) == 0 {
		val := provider.Constructor()
		if constructor, ok := val.(starlark.String); ok {
			if constructor.GoString() != identifiers.capvProvider {
				return starlark.None, errors.New("unknown capi provider")
			}
		}

		pathVal, err := provider.Attr("kubeconfig")
		if err != nil {
			return starlark.None, errors.Wrap(err, "could not find the kubeconfig attribute")
		}
		pathStr, ok := pathVal.(starlark.String)
		if !ok {
			return starlark.None, errors.New("could not fetch kubeconfig")
		}
		path = pathStr.GoString()
	}

	structVal := starlarkstruct.FromStringDict(starlarkstruct.Default, starlark.StringDict{
		"path": starlark.String(path),
	})

	// save dict to be used as default
	thread.SetLocal(identifiers.kubeCfg, structVal)

	return structVal, nil
}

// addDefaultKubeConf initializes a Starlark Dict with default
// KUBECONFIG configuration data
func addDefaultKubeConf(thread *starlark.Thread) error {
	args := []starlark.Tuple{
		{starlark.String("path"), starlark.String(defaults.kubeconfig)},
	}

	_, err := KubeConfigFn(thread, nil, nil, args)
	if err != nil {
		return err
	}

	return nil
}

// getKubeConfigPath is responsible to obtain the path to the kubeconfig
// It checks for the `path` key in the input args for the directive otherwise
// falls back to the default kube_config from the thread context
func getKubeConfigPath(thread *starlark.Thread, structVal *starlarkstruct.Struct) (string, error) {
	var (
		err   error
		kcVal starlark.Value
	)

	if kcVal, err = structVal.Attr("kube_config"); err != nil {
		kubeConfigData := thread.Local(identifiers.kubeCfg)
		kcVal = kubeConfigData.(starlark.Value)
	}

	kubeConfigVal, ok := kcVal.(*starlarkstruct.Struct)
	if !ok {
		return "", err
	}
	return getKubeConfigFromStruct(kubeConfigVal)
}

func getKubeConfigFromStruct(kubeConfigStructVal *starlarkstruct.Struct) (string, error) {
	kvPathVal, err := kubeConfigStructVal.Attr("path")
	if err != nil {
		return "", errors.Wrap(err, "failed to extract kubeconfig path")
	}
	kvPathStrVal, ok := kvPathVal.(starlark.String)
	if !ok {
		return "", errors.New("failed to extract management kubeconfig")
	}
	return kvPathStrVal.GoString(), nil
}
