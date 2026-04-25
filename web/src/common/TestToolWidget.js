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

import React from "react";
import {Button, Col, Row, Select} from "antd";
import * as Setting from "../Setting";
import i18next from "i18next";
import * as ProviderBackend from "../backend/ProviderBackend";
import {checkProvider} from "./ProviderWidget";
import Editor from "./Editor";
import ChatWidget from "./ChatWidget";

const {Option} = Select;

const OFFICE_TOOL_CONTENT = {
  "All": JSON.stringify({tool: "word_read", arguments: {path: "/path/to/document.docx"}}, null, 2),
  "Word Read": JSON.stringify({tool: "word_read", arguments: {path: "/path/to/document.docx"}}, null, 2),
  "Word Write": JSON.stringify({tool: "word_write", arguments: {path: "/path/to/output.docx", content: "Hello, World!\nThis is a new paragraph."}}, null, 2),
  "Excel Read": JSON.stringify({tool: "excel_read", arguments: {path: "/path/to/spreadsheet.xlsx", sheet: "Sheet1"}}, null, 2),
  "Excel Write": JSON.stringify({tool: "excel_write", arguments: {path: "/path/to/output.xlsx", data: "Name,Age\nAlice,30\nBob,25", sheet: "Sheet1"}}, null, 2),
  "PowerPoint Read": JSON.stringify({tool: "pptx_read", arguments: {path: "/path/to/presentation.pptx"}}, null, 2),
  "PowerPoint Write": JSON.stringify({tool: "pptx_write", arguments: {path: "/path/to/output.pptx", slides: ["Slide 1 title\nSlide 1 content", "Slide 2 title\nSlide 2 content"]}}, null, 2),
};

const DEFAULT_TOOL_CONTENT = {
  Time: JSON.stringify({tool: "time", arguments: {operation: "current", timezone: "Asia/Shanghai"}}, null, 2),
  "Web Search": JSON.stringify({tool: "web_search", arguments: {query: "Casibase web search", count: 3, language: "en", country: "us"}}, null, 2),
  Shell: JSON.stringify({tool: "shell", arguments: {command: "echo hello"}}, null, 2),
  "Web Fetch": JSON.stringify({tool: "web_fetch", arguments: {url: "https://casibase.org", max_length: 3000}}, null, 2),
  "Web Browser": JSON.stringify({tool: "web_browser", arguments: {url: "https://casibase.org", timeout: 60}}, null, 2),
};

function isValidToolTestJson(content) {
  try {
    const parsed = JSON.parse(content);
    return parsed && typeof parsed.tool === "string" && parsed.tool.trim() !== "";
  } catch (e) {
    return false;
  }
}

function buildDefaultToolTestJson(provider) {
  if (provider.type === "Office") {
    const subType = provider.subType || "All";
    return OFFICE_TOOL_CONTENT[subType] || OFFICE_TOOL_CONTENT["All"];
  }
  if (DEFAULT_TOOL_CONTENT[provider.type]) {
    return DEFAULT_TOOL_CONTENT[provider.type];
  }
  return JSON.stringify({tool: "", arguments: {}}, null, 2);
}

class TestToolWidget extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      testButtonLoading: false,
      testResult: "",
      modelProviders: [],
      modelProvidersLoading: false,
      // Track the last subType we synced so we can detect in-place mutations.
      // ProviderEditPage mutates the provider object reference rather than
      // replacing it, so provider !== prevProps.provider is always false.
      lastSyncedType: props.provider ? (props.provider.type || null) : null,
      lastSyncedSubType: props.provider ? (props.provider.subType || null) : null,
    };
  }

  componentDidMount() {
    this.syncFromProvider(this.props.provider);
    if (this.props.provider && this.props.provider.category === "Tool") {
      this.loadModelProviders();
    }
  }

  componentDidUpdate() {
    const {provider} = this.props;
    if (!provider || provider.category !== "Tool") {
      return;
    }

    // Detect type change and reset example content.
    const currentType = provider.type || null;
    if (currentType !== this.state.lastSyncedType) {
      // eslint-disable-next-line react/no-did-update-set-state
      this.setState({lastSyncedType: currentType, lastSyncedSubType: provider.subType || null, testResult: ""});
      if (this.props.onUpdateProvider) {
        this.props.onUpdateProvider("testContent", buildDefaultToolTestJson(provider));
      }
      return;
    }

    // Detect Office subType change via our own tracked state.
    if (provider.type === "Office") {
      const currentSubType = provider.subType || null;
      if (currentSubType !== this.state.lastSyncedSubType) {
        // eslint-disable-next-line react/no-did-update-set-state
        this.setState({lastSyncedSubType: currentSubType});
        if (this.props.onUpdateProvider) {
          this.props.onUpdateProvider("testContent", buildDefaultToolTestJson(provider));
        }
        return;
      }
    }

    if (this.state.modelProviders.length === 0 && !this.state.modelProvidersLoading) {
      this.loadModelProviders();
    }
  }

  loadModelProviders() {
    this.setState({modelProvidersLoading: true});
    ProviderBackend.getProviders("admin")
      .then((res) => {
        if (res.status === "ok") {
          this.setState({modelProviders: res.data.filter(p => p.category === "Model"), modelProvidersLoading: false});
        } else {
          this.setState({modelProvidersLoading: false});
        }
      });
  }

  syncFromProvider(provider, prevProvider) {
    const {onUpdateProvider} = this.props;
    if (!provider || provider.category !== "Tool") {
      return;
    }
    const needsDefault = !provider.testContent ||
      provider.testContent.trim() === "" ||
      !isValidToolTestJson(provider.testContent);
    if (needsDefault && onUpdateProvider) {
      onUpdateProvider("testContent", buildDefaultToolTestJson(provider));
    }
    const prevSummary = prevProvider ? prevProvider.resultSummary : null;
    if (provider.resultSummary && provider.resultSummary !== prevSummary) {
      this.setState({testResult: provider.resultSummary});
    }
  }

  async sendTestTool(provider, originalProvider) {
    let parsed;
    try {
      parsed = JSON.parse(provider.testContent);
    } catch (e) {
      Setting.showMessage("error", `${i18next.t("provider:Invalid tool test JSON")}: ${e.message}`);
      return;
    }
    if (!parsed || typeof parsed.tool !== "string" || parsed.tool.trim() === "") {
      Setting.showMessage("error", i18next.t("provider:Tool test JSON must include tool"));
      return;
    }

    await checkProvider(provider, originalProvider);
    this.setState({testButtonLoading: true, testResult: ""});

    try {
      const res = await ProviderBackend.testToolProvider(provider);
      if (res.status === "ok") {
        let out;
        if (typeof res.data === "string") {
          try {
            out = JSON.stringify(JSON.parse(res.data), null, 2);
          } catch (e) {
            out = res.data;
          }
        } else {
          out = JSON.stringify(res.data, null, 2);
        }
        this.setState({testResult: out});
        Setting.showMessage("success", i18next.t("general:Success"));
        if (this.props.onUpdateProvider) {
          this.props.onUpdateProvider("resultSummary", out);
          this.props.onUpdateProvider("errorText", "");
        }
        await ProviderBackend.updateProvider(provider.owner, provider.name, {...provider, resultSummary: out, errorText: ""});
      } else {
        Setting.showMessage("error", res.msg || i18next.t("general:Failed to save"));
      }
    } catch (error) {
      Setting.showMessage("error", `${i18next.t("general:Failed to connect to server")}: ${error.message}`);
    } finally {
      this.setState({testButtonLoading: false});
    }
  }

  render() {
    const {provider, originalProvider, onUpdateProvider, account} = this.props;
    const {modelProviders} = this.state;
    const selectedModelProvider = provider.modelProvider || "";

    if (!provider || provider.category !== "Tool") {
      return null;
    }

    return (
      <React.Fragment>
        <Row style={{marginTop: "20px"}} >
          <Col style={{marginTop: "5px"}} span={(Setting.isMobile()) ? 22 : 2}>
            {Setting.getLabel(i18next.t("provider:Provider test"), i18next.t("provider:Tool test JSON - Tooltip"))} :
          </Col>
          <Col span={10} >
            <Editor
              value={provider.testContent}
              lang="json"
              height="150px"
              dark
              onChange={value => {onUpdateProvider("testContent", value);}}
            />
          </Col>
          <Col span={6} >
            <Button
              style={{marginLeft: "10px", marginBottom: "5px"}}
              type="primary"
              loading={this.state.testButtonLoading}
              disabled={!provider.testContent || provider.testContent.trim() === ""}
              onClick={() => this.sendTestTool(provider, originalProvider)}
            >
              {i18next.t("provider:Invoke tool")}
            </Button>
          </Col>
        </Row>
        <Row style={{marginTop: "10px"}}>
          <Col span={2}></Col>
          <Col span={10}>
            <div style={{marginBottom: "5px"}}><strong>{i18next.t("provider:Tool result")}:</strong></div>
            <Editor
              value={this.state.testResult}
              lang="json"
              height="150px"
              dark
              readOnly
            />
          </Col>
        </Row>
        <Row style={{marginTop: "20px"}}>
          <Col style={{marginTop: "5px"}} span={(Setting.isMobile()) ? 22 : 2}>
            {Setting.getLabel(i18next.t("provider:Chat test"), i18next.t("provider:Chat test - Tooltip"))} :
          </Col>
          <Col span={20}>
            <Row style={{marginBottom: "10px"}}>
              <Col span={24}>
                <Select
                  style={{width: "100%"}}
                  placeholder={i18next.t("provider:Select model provider")}
                  value={selectedModelProvider || undefined}
                  onChange={(value) => onUpdateProvider("modelProvider", value)}
                  showSearch
                  filterOption={(input, option) =>
                    option.children[1].toLowerCase().includes(input.toLowerCase())
                  }
                >
                  {modelProviders.map((mp, index) => (
                    <Option key={index} value={mp.name}>
                      <img width={20} height={20} style={{marginBottom: "3px", marginRight: "10px"}}
                        src={Setting.getProviderLogoURL({category: mp.category, type: mp.type})}
                        alt={mp.name} />
                      {mp.displayName || mp.name}
                    </Option>
                  ))}
                </Select>
              </Col>
            </Row>
            {selectedModelProvider ? (
              <ChatWidget
                key={`${provider.name}-${selectedModelProvider}`}
                chatName={`chat_tool_${provider.name}`}
                displayName={`${provider.displayName || provider.name} - Chat Test`}
                category="ToolTest"
                modelProvider={selectedModelProvider}
                toolProvider={provider.name}
                account={account}
                height="600px"
                showHeader={true}
                showNewChatButton={true}
              />
            ) : (
              <div style={{
                width: "100%",
                height: "100px",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                border: "1px solid #d9d9d9",
                borderRadius: "6px",
                color: "#999",
              }}>
                {i18next.t("provider:Please select a model provider first")}
              </div>
            )}
          </Col>
        </Row>
      </React.Fragment>
    );
  }
}

export default TestToolWidget;
