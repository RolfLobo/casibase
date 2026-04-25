// Copyright 2026 The Casibase Authors. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tool

import (
	"fmt"

	"github.com/casibase/casibase/agent/builtin_tool"
	"github.com/casibase/casibase/i18n"
)

// Provider supplies LLM-callable tools (object.Provider category "Tool").
type Provider interface {
	BuiltinTools() []builtin_tool.BuiltinTool
}

// ProviderConfig contains the Provider fields needed to construct builtin tools.
type ProviderConfig struct {
	Category     string
	Type         string
	SubType      string
	ProviderUrl  string
	ClientId     string
	ClientSecret string
	EnableProxy  bool
}

// NewProvider instantiates a Tool provider implementation from category and type.
func NewProvider(config ProviderConfig, lang string) (Provider, error) {
	if config.Category != "Tool" {
		return nil, fmt.Errorf(i18n.Translate(lang, "tool:expected category Tool, got %s"), config.Category)
	}
	switch config.Type {
	case "Time":
		return &TimeProvider{}, nil
	case "Web Search":
		return NewWebSearchProvider(config)
	case "Shell":
		return &ShellProvider{}, nil
	case "Office":
		return &OfficeProvider{subType: officeSubType(config.SubType)}, nil
	case "Web Fetch":
		return NewWebFetchProvider(config)
	case "Web Browser":
		return NewBrowserProvider(config)
	default:
		return nil, fmt.Errorf(i18n.Translate(lang, "tool:unsupported tool provider type: %s"), config.Type)
	}
}
