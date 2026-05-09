// Copyright 2026 The OpenAgent Authors. All Rights Reserved.
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

import React, {useState} from "react";
import {Alert, Button, Col, Input, Row, Select, Space, Tag, Typography} from "antd";
import {CheckCircleFilled, LinkOutlined} from "@ant-design/icons";
import * as ProviderBackend from "./backend/ProviderBackend";
import * as PipeBackend from "./backend/PipeBackend";
import * as Setting from "./Setting";
import i18next from "i18next";
import moment from "moment";

const {Title, Text, Paragraph} = Typography;
const {Option} = Select;

const MODEL_PROVIDERS = [
  {
    id: "OpenAI",
    label: "OpenAI",
    desc: "GPT-4o, GPT-4, o3...",
    needsApiKey: true,
    needsUrl: false,
    defaultSubType: "gpt-4o",
    models: ["gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-4", "gpt-3.5-turbo", "o1", "o3-mini"],
  },
  {
    id: "Claude",
    label: "Claude",
    desc: "Claude Sonnet, Opus...",
    needsApiKey: true,
    needsUrl: false,
    defaultSubType: "claude-sonnet-4-5",
    models: ["claude-opus-4-5", "claude-sonnet-4-5", "claude-3-5-sonnet-20241022", "claude-3-opus-20240229", "claude-3-haiku-20240307"],
  },
  {
    id: "Gemini",
    label: "Gemini",
    desc: "Gemini 2.0, 1.5 Pro...",
    needsApiKey: true,
    needsUrl: false,
    defaultSubType: "gemini-2.0-flash",
    models: ["gemini-2.0-flash", "gemini-1.5-pro", "gemini-1.5-flash", "gemini-pro"],
  },
  {
    id: "DeepSeek",
    label: "DeepSeek",
    desc: "DeepSeek-V3, R1...",
    needsApiKey: true,
    needsUrl: false,
    defaultSubType: "deepseek-chat",
    models: ["deepseek-chat", "deepseek-reasoner"],
  },
  {
    id: "Grok",
    label: "Grok",
    desc: "Grok-3, Grok-2...",
    needsApiKey: true,
    needsUrl: false,
    defaultSubType: "grok-3",
    models: ["grok-3", "grok-3-mini", "grok-2"],
  },
  {
    id: "Ollama",
    label: "Ollama",
    desc: i18next.t("setup:Run models locally"),
    needsApiKey: false,
    needsUrl: true,
    defaultSubType: "llama3",
    models: [],
    urlPlaceholder: "http://localhost:11434",
    defaultUrl: "http://localhost:11434",
  },
  {
    id: "OpenRouter",
    label: "OpenRouter",
    desc: "100+ models unified",
    needsApiKey: true,
    needsUrl: false,
    defaultSubType: "openai/gpt-4o",
    models: ["openai/gpt-4o", "anthropic/claude-3.5-sonnet", "google/gemini-pro-1.5", "meta-llama/llama-3.1-70b-instruct"],
  },
  {
    id: "Mistral",
    label: "Mistral",
    desc: "Mistral Large, Medium...",
    needsApiKey: true,
    needsUrl: false,
    defaultSubType: "mistral-large-latest",
    models: ["mistral-large-latest", "mistral-medium-latest", "mistral-small-latest", "codestral-latest"],
  },
  {
    id: "Azure",
    label: "Azure OpenAI",
    desc: "Azure-hosted GPT models",
    needsApiKey: true,
    needsUrl: true,
    defaultSubType: "gpt-4o",
    models: ["gpt-4o", "gpt-4-turbo", "gpt-4", "gpt-35-turbo"],
    urlPlaceholder: "https://your-resource.openai.azure.com",
  },
  {
    id: "OpenAI Compatible",
    label: "OpenAI Compatible",
    desc: "Any compatible API",
    needsApiKey: true,
    needsUrl: true,
    defaultSubType: "",
    models: [],
    urlPlaceholder: "https://api.example.com/v1",
  },
  {
    id: "Alibaba Cloud",
    label: "Alibaba Cloud",
    desc: "Qwen-Max, Qwen-Plus...",
    needsApiKey: true,
    needsUrl: false,
    defaultSubType: "qwen-max",
    models: ["qwen-max", "qwen-plus", "qwen-turbo", "qwen-long"],
  },
  {
    id: "Moonshot",
    label: "Moonshot (Kimi)",
    desc: "Kimi long-context models",
    needsApiKey: true,
    needsUrl: false,
    defaultSubType: "moonshot-v1-32k",
    models: ["moonshot-v1-8k", "moonshot-v1-32k", "moonshot-v1-128k"],
  },
  {
    id: "Silicon Flow",
    label: "Silicon Flow",
    desc: "DeepSeek, Qwen, and more",
    needsApiKey: true,
    needsUrl: false,
    defaultSubType: "deepseek-ai/DeepSeek-V3",
    models: ["deepseek-ai/DeepSeek-V3", "deepseek-ai/DeepSeek-R1", "Qwen/Qwen2.5-72B-Instruct"],
  },
  {
    id: "Volcano Engine",
    label: "Volcano Engine",
    desc: "ByteDance AI platform",
    needsApiKey: true,
    needsUrl: true,
    defaultSubType: "",
    models: [],
    urlPlaceholder: "your-endpoint-id",
  },
  {
    id: "Baidu Cloud",
    label: "Baidu Cloud",
    desc: "ERNIE Bot models",
    needsApiKey: true,
    needsUrl: false,
    defaultSubType: "ERNIE-4.0-8K",
    models: ["ERNIE-4.0-8K", "ERNIE-3.5-8K", "ERNIE-Speed-128K"],
  },
  {
    id: "Amazon Bedrock",
    label: "Amazon Bedrock",
    desc: "Claude, Llama on AWS",
    needsApiKey: true,
    needsClientId: true,
    needsUrl: false,
    needsRegion: true,
    defaultSubType: "anthropic.claude-3-5-sonnet-20241022-v2:0",
    models: ["anthropic.claude-3-5-sonnet-20241022-v2:0", "meta.llama3-70b-instruct-v1:0"],
  },
];

const PIPE_PLATFORMS = [
  {
    id: "Telegram",
    label: "Telegram",
    desc: "Connect via Telegram bot",
    tokenLabel: "Bot Token",
    tokenPlaceholder: "1234567890:ABCdefGHIjklMNOpqrsTUVwxyz",
    helpUrl: "https://core.telegram.org/bots#how-do-i-create-a-bot",
  },
  {
    id: "Discord",
    label: "Discord",
    desc: "Connect via Discord bot",
    tokenLabel: "Bot Token",
    tokenPlaceholder: "MTxxxxxx.Gyyyyy.zzzzzzzzzzz",
    helpUrl: "https://discord.com/developers/applications",
  },
  {
    id: "WhatsApp",
    label: "WhatsApp",
    desc: "Connect via WhatsApp Business",
    tokenLabel: "Access Token",
    tokenPlaceholder: "EAAxxxxxxxx...",
    helpUrl: "https://developers.facebook.com/docs/whatsapp",
  },
];

function ProviderCard({provider, selected, onClick}) {
  const logo = Setting.getProviderLogoURL({category: "Model", type: provider.id});
  const borderColor = selected ? "var(--ant-color-primary)" : "var(--ant-color-border)";
  const bgColor = selected ? "var(--ant-color-primary-bg)" : "var(--ant-color-bg-container)";

  return (
    <div
      onClick={onClick}
      style={{
        border: `2px solid ${borderColor}`,
        borderRadius: 12,
        padding: "14px 12px",
        cursor: "pointer",
        background: bgColor,
        position: "relative",
        textAlign: "center",
        transition: "all 0.2s",
        userSelect: "none",
        minWidth: 120,
      }}
    >
      {selected && (
        <CheckCircleFilled
          style={{
            position: "absolute",
            top: 6,
            right: 6,
            color: "var(--ant-color-primary)",
            fontSize: 16,
          }}
        />
      )}
      <div style={{height: 44, display: "flex", alignItems: "center", justifyContent: "center", marginBottom: 8}}>
        {logo ? (
          <img src={logo} alt={provider.label} style={{maxWidth: 44, maxHeight: 44, objectFit: "contain"}} />
        ) : (
          <div style={{width: 44, height: 44, borderRadius: 8, background: "var(--ant-color-fill-secondary)", display: "flex", alignItems: "center", justifyContent: "center", fontSize: 18, fontWeight: 700, color: "var(--ant-color-text-secondary)"}}>
            {provider.label[0]}
          </div>
        )}
      </div>
      <div style={{fontWeight: 600, fontSize: 13, marginBottom: 2, lineHeight: 1.3}}>{provider.label}</div>
      <div style={{fontSize: 11, color: "var(--ant-color-text-secondary)", lineHeight: 1.3}}>{provider.desc}</div>
    </div>
  );
}

function PipeCard({platform, selected, onClick}) {
  const logo = Setting.getProviderLogoURL({category: "Chat", type: platform.id});
  const borderColor = selected ? "var(--ant-color-primary)" : "var(--ant-color-border)";
  const bgColor = selected ? "var(--ant-color-primary-bg)" : "var(--ant-color-bg-container)";

  return (
    <div
      onClick={onClick}
      style={{
        border: `2px solid ${borderColor}`,
        borderRadius: 12,
        padding: "14px 12px",
        cursor: "pointer",
        background: bgColor,
        position: "relative",
        textAlign: "center",
        transition: "all 0.2s",
        userSelect: "none",
        minWidth: 120,
      }}
    >
      {selected && (
        <CheckCircleFilled
          style={{
            position: "absolute",
            top: 6,
            right: 6,
            color: "var(--ant-color-primary)",
            fontSize: 16,
          }}
        />
      )}
      <div style={{height: 44, display: "flex", alignItems: "center", justifyContent: "center", marginBottom: 8}}>
        {logo ? (
          <img src={logo} alt={platform.label} style={{maxWidth: 44, maxHeight: 44, objectFit: "contain"}} />
        ) : (
          <div style={{width: 44, height: 44, borderRadius: 8, background: "var(--ant-color-fill-secondary)", display: "flex", alignItems: "center", justifyContent: "center", fontSize: 18, fontWeight: 700, color: "var(--ant-color-text-secondary)"}}>
            {platform.label[0]}
          </div>
        )}
      </div>
      <div style={{fontWeight: 600, fontSize: 13, marginBottom: 2, lineHeight: 1.3}}>{platform.label}</div>
      <div style={{fontSize: 11, color: "var(--ant-color-text-secondary)", lineHeight: 1.3}}>{platform.desc}</div>
    </div>
  );
}

function SectionTitle({number, title, subtitle}) {
  return (
    <div style={{marginBottom: 20}}>
      <div style={{display: "flex", alignItems: "center", gap: 10, marginBottom: 4}}>
        <div style={{
          width: 28,
          height: 28,
          borderRadius: "50%",
          background: "var(--ant-color-primary)",
          color: "#fff",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          fontWeight: 700,
          fontSize: 14,
          flexShrink: 0,
        }}>{number}</div>
        <span style={{fontSize: 18, fontWeight: 700}}>{title}</span>
        {subtitle && <Tag color="default" style={{marginLeft: 4, fontWeight: 400}}>{subtitle}</Tag>}
      </div>
    </div>
  );
}

function FieldRow({label, children, hint}) {
  return (
    <div style={{marginBottom: 16}}>
      <div style={{marginBottom: 6, fontWeight: 500, fontSize: 14}}>{label}</div>
      {children}
      {hint && <div style={{marginTop: 4, fontSize: 12, color: "var(--ant-color-text-secondary)"}}>{hint}</div>}
    </div>
  );
}

function QuickSetupPage({account}) {
  const [selectedModelType, setSelectedModelType] = useState(null);
  const [providerName, setProviderName] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [clientId, setClientId] = useState("");
  const [region, setRegion] = useState("us-east-1");
  const [subType, setSubType] = useState("");
  const [providerUrl, setProviderUrl] = useState("");

  const [selectedPipeType, setSelectedPipeType] = useState(null);
  const [pipeSkipped, setPipeSkipped] = useState(false);
  const [pipeName, setPipeName] = useState("");
  const [pipeToken, setPipeToken] = useState("");

  const [saving, setSaving] = useState(false);
  const [savedProvider, setSavedProvider] = useState(null);
  const [savedPipe, setSavedPipe] = useState(null);

  const selectedModelProvider = MODEL_PROVIDERS.find(p => p.id === selectedModelType);
  const selectedPipePlatform = PIPE_PLATFORMS.find(p => p.id === selectedPipeType);

  function handleSelectModel(provider) {
    if (selectedModelType === provider.id) {return;}
    setSelectedModelType(provider.id);
    const rand = Setting.getRandomName();
    setProviderName(`provider_${rand}`);
    setApiKey("");
    setClientId("");
    setRegion("us-east-1");
    setSubType(provider.defaultSubType || "");
    setProviderUrl(provider.defaultUrl || "");
  }

  function handleSelectPipe(platform) {
    if (selectedPipeType === platform.id) {return;}
    setSelectedPipeType(platform.id);
    setPipeSkipped(false);
    const rand = Setting.getRandomName();
    setPipeName(`pipe_${rand}`);
    setPipeToken("");
  }

  function handleSkipPipe() {
    setSelectedPipeType(null);
    setPipeSkipped(true);
  }

  async function handleSave() {
    if (!selectedModelType) {
      Setting.showMessage("error", i18next.t("setup:Please choose an AI model provider"));
      return;
    }
    if (!providerName.trim()) {
      Setting.showMessage("error", i18next.t("setup:Provider name is required"));
      return;
    }
    const mp = selectedModelProvider;
    if (mp.needsApiKey && !apiKey.trim()) {
      Setting.showMessage("error", i18next.t("setup:API Key is required"));
      return;
    }
    if (mp.needsUrl && !providerUrl.trim()) {
      Setting.showMessage("error", i18next.t("setup:URL is required"));
      return;
    }
    if (!subType.trim()) {
      Setting.showMessage("error", i18next.t("setup:Model name is required"));
      return;
    }
    if (selectedPipeType && !pipeSkipped) {
      if (!pipeToken.trim()) {
        Setting.showMessage("error", i18next.t("setup:Token is required"));
        return;
      }
    }

    setSaving(true);

    const provider = {
      owner: "admin",
      name: providerName.trim(),
      createdTime: moment().format(),
      displayName: `${mp.label} (${subType})`,
      displayName2: "",
      category: "Model",
      type: mp.id,
      subType: subType.trim(),
      clientId: mp.needsClientId ? clientId.trim() : "",
      clientSecret: apiKey.trim(),
      mcpTools: [],
      enableThinking: false,
      temperature: 1,
      topP: 1,
      topK: 4,
      frequencyPenalty: 0,
      presencePenalty: 0,
      inputPricePerThousandTokens: 0,
      outputPricePerThousandTokens: 0,
      currency: "USD",
      providerUrl: mp.needsUrl ? providerUrl.trim() : "",
      apiVersion: "",
      apiKey: "",
      network: mp.needsRegion ? region.trim() : "",
      userKey: "",
      userCert: "",
      signKey: "",
      signCert: "",
      compatibleProvider: "",
      contractName: "",
      contractMethod: "",
      state: "Active",
      isRemote: false,
    };

    try {
      const res = await ProviderBackend.addProvider(provider);
      if (res.status !== "ok") {
        Setting.showMessage("error", `${i18next.t("general:Failed to add")}: ${res.msg}`);
        setSaving(false);
        return;
      }
      setSavedProvider(provider);

      if (selectedPipeType && !pipeSkipped && pipeToken.trim()) {
        const pipe = {
          owner: "admin",
          name: pipeName.trim(),
          createdTime: moment().format(),
          displayName: `${selectedPipePlatform.label} Bot`,
          type: selectedPipeType,
          token: pipeToken.trim(),
          secretKey: "",
          domain: "",
          isDefault: false,
          state: "Active",
        };
        const pipeRes = await PipeBackend.addPipe(pipe);
        if (pipeRes.status !== "ok") {
          Setting.showMessage("error", `${i18next.t("general:Failed to add")} pipe: ${pipeRes.msg}`);
          setSaving(false);
          return;
        }
        setSavedPipe(pipe);
      }

      Setting.showMessage("success", i18next.t("setup:Configuration saved successfully"));
    } catch (err) {
      Setting.showMessage("error", String(err));
    } finally {
      setSaving(false);
    }
  }

  const cardGridStyle = {
    display: "grid",
    gridTemplateColumns: "repeat(auto-fill, minmax(130px, 1fr))",
    gap: 12,
    marginBottom: 24,
  };

  const sectionStyle = {
    background: "var(--ant-color-bg-container)",
    border: "1px solid var(--ant-color-border)",
    borderRadius: 16,
    padding: "28px 28px 20px",
    marginBottom: 24,
  };

  if (savedProvider) {
    return (
      <div style={{maxWidth: 720, margin: "0 auto", padding: "32px 16px"}}>
        <Alert
          type="success"
          showIcon
          message={<span style={{fontSize: 16, fontWeight: 600}}>{i18next.t("setup:Setup complete!")}</span>}
          description={
            <div style={{marginTop: 8}}>
              <Paragraph>{i18next.t("setup:Your configuration has been saved. You can now start chatting or further customize your setup.")}</Paragraph>
              <Space wrap>
                <Button type="primary" href="/chat">{i18next.t("general:Chat")}</Button>
                <Button href={`/providers/${savedProvider.name}`} icon={<LinkOutlined />}>
                  {i18next.t("setup:View Provider")}
                </Button>
                {savedPipe && (
                  <Button href={`/pipes/${savedPipe.name}`} icon={<LinkOutlined />}>
                    {i18next.t("setup:View Pipe")}
                  </Button>
                )}
                <Button onClick={() => {
                  setSavedProvider(null);
                  setSavedPipe(null);
                  setSelectedModelType(null);
                  setSelectedPipeType(null);
                  setPipeSkipped(false);
                }}>
                  {i18next.t("setup:Setup another")}
                </Button>
              </Space>
            </div>
          }
          style={{borderRadius: 16, marginBottom: 24}}
        />

        <div style={sectionStyle}>
          <div style={{fontWeight: 600, fontSize: 15, marginBottom: 12}}>{i18next.t("setup:Created Resources")}</div>
          <div style={{display: "flex", flexDirection: "column", gap: 8}}>
            <div style={{display: "flex", alignItems: "center", gap: 8}}>
              <Tag color="blue">{i18next.t("general:Providers")}</Tag>
              <Text code>{savedProvider.name}</Text>
              <Text type="secondary">({savedProvider.type} / {savedProvider.subType})</Text>
            </div>
            {savedPipe && (
              <div style={{display: "flex", alignItems: "center", gap: 8}}>
                <Tag color="green">{i18next.t("general:Pipes")}</Tag>
                <Text code>{savedPipe.name}</Text>
                <Text type="secondary">({savedPipe.type})</Text>
              </div>
            )}
          </div>
        </div>
      </div>
    );
  }

  return (
    <div style={{maxWidth: 800, margin: "0 auto", padding: "32px 16px"}}>
      <div style={{marginBottom: 32}}>
        <Title level={3} style={{marginBottom: 4}}>{i18next.t("general:Quick Setup")}</Title>
        <Text type="secondary" style={{fontSize: 15}}>{i18next.t("setup:Get your AI up and running in minutes — no technical knowledge required.")}</Text>
      </div>

      {/* Section 1: Model Provider */}
      <div style={sectionStyle}>
        <SectionTitle number={1} title={i18next.t("setup:Choose an AI Model Provider")} />
        <div style={cardGridStyle}>
          {MODEL_PROVIDERS.map(p => (
            <ProviderCard
              key={p.id}
              provider={p}
              selected={selectedModelType === p.id}
              onClick={() => handleSelectModel(p)}
            />
          ))}
        </div>

        {selectedModelProvider && (
          <div style={{borderTop: "1px solid var(--ant-color-border)", paddingTop: 20, marginTop: 4}}>
            <div style={{fontWeight: 600, fontSize: 15, marginBottom: 16}}>
              {i18next.t("setup:Configure")} {selectedModelProvider.label}
            </div>

            <Row gutter={16}>
              <Col xs={24} md={12}>
                <FieldRow label={i18next.t("general:Model")}>
                  {selectedModelProvider.models.length > 0 ? (
                    <Select
                      value={subType}
                      onChange={setSubType}
                      style={{width: "100%"}}
                      showSearch
                      allowClear={false}
                    >
                      {selectedModelProvider.models.map(m => (
                        <Option key={m} value={m}>{m}</Option>
                      ))}
                    </Select>
                  ) : (
                    <Input
                      value={subType}
                      onChange={e => setSubType(e.target.value)}
                      placeholder={i18next.t("setup:Enter model name")}
                    />
                  )}
                </FieldRow>
              </Col>

              <Col xs={24} md={12}>
                <FieldRow label={i18next.t("setup:Internal Name")} hint={i18next.t("setup:Used internally to identify this provider")}>
                  <Input
                    value={providerName}
                    onChange={e => setProviderName(e.target.value)}
                  />
                </FieldRow>
              </Col>
            </Row>

            {selectedModelProvider.needsClientId && (
              <Row gutter={16}>
                <Col xs={24} md={12}>
                  <FieldRow label={selectedModelProvider.id === "Amazon Bedrock" ? "Access Key ID" : "Client ID"}>
                    <Input
                      value={clientId}
                      onChange={e => setClientId(e.target.value)}
                      placeholder="AKIA..."
                    />
                  </FieldRow>
                </Col>
              </Row>
            )}

            {selectedModelProvider.needsApiKey && (
              <FieldRow
                label={i18next.t("setup:API Key")}
                hint={i18next.t("setup:Your secret API key from the provider dashboard")}
              >
                <Input.Password
                  value={apiKey}
                  onChange={e => setApiKey(e.target.value)}
                  placeholder="sk-..."
                  size="large"
                />
              </FieldRow>
            )}

            {selectedModelProvider.needsRegion && (
              <Row gutter={16}>
                <Col xs={24} md={12}>
                  <FieldRow label={i18next.t("general:Region")}>
                    <Input
                      value={region}
                      onChange={e => setRegion(e.target.value)}
                      placeholder="us-east-1"
                    />
                  </FieldRow>
                </Col>
              </Row>
            )}

            {selectedModelProvider.needsUrl && (
              <FieldRow
                label={selectedModelProvider.id === "Ollama" ? i18next.t("setup:Ollama Server URL") : i18next.t("setup:API Endpoint URL")}
                hint={
                  selectedModelProvider.id === "Ollama"
                    ? i18next.t("setup:Make sure Ollama is running locally before saving")
                    : undefined
                }
              >
                <Input
                  value={providerUrl}
                  onChange={e => setProviderUrl(e.target.value)}
                  placeholder={selectedModelProvider.urlPlaceholder || "https://"}
                  prefix={<LinkOutlined />}
                  size="large"
                />
              </FieldRow>
            )}
          </div>
        )}
      </div>

      {/* Section 2: Pipe (optional) */}
      <div style={sectionStyle}>
        <SectionTitle
          number={2}
          title={i18next.t("setup:Connect a Messaging Platform")}
          subtitle={i18next.t("setup:Optional")}
        />
        <Paragraph type="secondary" style={{marginBottom: 16}}>
          {i18next.t("setup:Connect a messaging app so users can chat with your AI through Telegram, Discord, or WhatsApp. You can skip this step and set it up later.")}
        </Paragraph>

        <div style={{...cardGridStyle, gridTemplateColumns: "repeat(auto-fill, minmax(140px, 1fr))"}}>
          {PIPE_PLATFORMS.map(p => (
            <PipeCard
              key={p.id}
              platform={p}
              selected={selectedPipeType === p.id}
              onClick={() => handleSelectPipe(p)}
            />
          ))}
          <div
            onClick={handleSkipPipe}
            style={{
              border: `2px solid ${pipeSkipped ? "var(--ant-color-border)" : "var(--ant-color-border)"}`,
              borderRadius: 12,
              padding: "14px 12px",
              cursor: "pointer",
              background: pipeSkipped ? "var(--ant-color-fill-quaternary)" : "var(--ant-color-bg-container)",
              textAlign: "center",
              transition: "all 0.2s",
              userSelect: "none",
              display: "flex",
              flexDirection: "column",
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            <div style={{fontSize: 28, marginBottom: 8, opacity: 0.4}}>✕</div>
            <div style={{fontWeight: 600, fontSize: 13, marginBottom: 2}}>{i18next.t("setup:Skip")}</div>
            <div style={{fontSize: 11, color: "var(--ant-color-text-secondary)"}}>{i18next.t("setup:Set up later")}</div>
          </div>
        </div>

        {selectedPipePlatform && !pipeSkipped && (
          <div style={{borderTop: "1px solid var(--ant-color-border)", paddingTop: 20, marginTop: 4}}>
            <div style={{fontWeight: 600, fontSize: 15, marginBottom: 16}}>
              {i18next.t("setup:Configure")} {selectedPipePlatform.label}
            </div>

            <Row gutter={16}>
              <Col xs={24} md={12}>
                <FieldRow label={i18next.t("setup:Internal Name")}>
                  <Input
                    value={pipeName}
                    onChange={e => setPipeName(e.target.value)}
                  />
                </FieldRow>
              </Col>
            </Row>

            <FieldRow
              label={selectedPipePlatform.tokenLabel}
              hint={
                <span>
                  {i18next.t("setup:How to get a token?")}&nbsp;
                  <a href={selectedPipePlatform.helpUrl} target="_blank" rel="noreferrer">
                    {i18next.t("setup:View guide")} <LinkOutlined />
                  </a>
                </span>
              }
            >
              <Input.Password
                value={pipeToken}
                onChange={e => setPipeToken(e.target.value)}
                placeholder={selectedPipePlatform.tokenPlaceholder}
                size="large"
              />
            </FieldRow>
          </div>
        )}
      </div>

      {/* Save Button */}
      <div style={{display: "flex", justifyContent: "flex-end", gap: 12}}>
        <Button
          type="primary"
          size="large"
          loading={saving}
          disabled={!selectedModelType}
          onClick={handleSave}
          style={{minWidth: 160, borderRadius: 10, height: 44, fontWeight: 600}}
        >
          {saving ? i18next.t("setup:Saving...") : i18next.t("setup:Save Configuration")}
        </Button>
      </div>
    </div>
  );
}

export default QuickSetupPage;
