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
import {Button, Col, Input, Row} from "antd";
import * as Setting from "../Setting";
import i18next from "i18next";
import * as ProviderBackend from "../backend/ProviderBackend";
import {checkProvider} from "./ProviderWidget";

const {TextArea} = Input;

function buildDefaultMcpTestJson(provider) {
  if (provider.mcpTools && provider.mcpTools.length > 0) {
    const mt = provider.mcpTools.find(t => t.isEnabled !== false) || provider.mcpTools[0];
    try {
      const tools = JSON.parse(mt.tools || "[]");
      if (tools.length > 0 && tools[0].name) {
        const toolId = `${mt.serverName}__${tools[0].name}`;
        return JSON.stringify({tool: toolId, arguments: {}}, null, 2);
      }
    } catch (e) {
      // ignore parse errors, fall through
    }
  }
  return "{\n  \"tool\": \"serverName__toolName\",\n  \"arguments\": {}\n}";
}

class TestMcpWidget extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      testButtonLoading: false,
      testResult: "",
    };
  }

  componentDidMount() {
    const {provider, onUpdateProvider} = this.props;
    if (provider && provider.category === "Agent" && provider.type === "MCP") {
      if (!provider.testContent || provider.testContent.trim() === "") {
        const def = buildDefaultMcpTestJson(provider);
        if (onUpdateProvider) {
          onUpdateProvider("testContent", def);
        }
      }
    }
  }

  async sendTestMcp(provider, originalProvider) {
    let parsed;
    try {
      parsed = JSON.parse(provider.testContent);
    } catch (e) {
      Setting.showMessage("error", `${i18next.t("provider:Invalid MCP test JSON")}: ${e.message}`);
      return;
    }
    if (!parsed || typeof parsed.tool !== "string" || parsed.tool.trim() === "") {
      Setting.showMessage("error", i18next.t("provider:MCP test JSON must include tool"));
      return;
    }

    await checkProvider(provider, originalProvider);
    this.setState({testButtonLoading: true, testResult: ""});

    try {
      const res = await ProviderBackend.testMcpProvider(provider);
      if (res.status === "ok") {
        const out = typeof res.data === "string" ? res.data : JSON.stringify(res.data, null, 2);
        this.setState({testResult: out});
        Setting.showMessage("success", i18next.t("general:Success"));
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
    const {provider, originalProvider, onUpdateProvider} = this.props;

    if (!provider || provider.category !== "Agent" || provider.type !== "MCP") {
      return null;
    }

    return (
      <React.Fragment>
        <Row style={{marginTop: "20px"}} >
          <Col style={{marginTop: "5px"}} span={(Setting.isMobile()) ? 22 : 2}>
            {Setting.getLabel(i18next.t("provider:Provider test"), i18next.t("provider:MCP test JSON - Tooltip"))} :
          </Col>
          <Col span={10} >
            <Input.TextArea
              rows={4}
              value={provider.testContent}
              placeholder='{"tool":"server__toolName","arguments":{}}'
              onChange={e => {onUpdateProvider("testContent", e.target.value);}}
            />
          </Col>
          <Col span={6} >
            <Button
              style={{marginLeft: "10px", marginBottom: "5px"}}
              type="primary"
              loading={this.state.testButtonLoading}
              disabled={!provider.testContent || provider.testContent.trim() === ""}
              onClick={() => this.sendTestMcp(provider, originalProvider)}
            >
              {i18next.t("provider:Invoke MCP tool")}
            </Button>
          </Col>
        </Row>
        {this.state.testResult ? (
          <Row style={{marginTop: "10px"}}>
            <Col span={2}></Col>
            <Col span={20}>
              <div style={{border: "1px solid #d9d9d9", borderRadius: "6px", padding: "10px", backgroundColor: "#fafafa"}}>
                <div><strong>{i18next.t("provider:MCP tool result")}:</strong></div>
                <TextArea autoSize={{minRows: 4, maxRows: 16}} value={this.state.testResult} readOnly style={{marginTop: "5px", fontFamily: "monospace", fontSize: "12px"}} />
              </div>
            </Col>
          </Row>
        ) : null}
      </React.Fragment>
    );
  }
}

export default TestMcpWidget;
