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

import React from "react";
import {Alert, Card, Col, Empty, List, Row, Spin, Statistic, Table, Tag, Typography} from "antd";
import {CheckCircleFilled, CloseCircleFilled, ExclamationCircleFilled, MessageOutlined, SafetyCertificateOutlined, StopOutlined, TeamOutlined, WarningOutlined} from "@ant-design/icons";
import ReactEcharts from "echarts-for-react";
import i18next from "i18next";
import * as AnalysisBackend from "./backend/AnalysisBackend";
import UserLabel from "./common/UserLabel";

const {Text} = Typography;

// shadcn/ui-style status palette — emerald/amber/red-500, matching shadcn's
// default `--destructive` token and the emerald/amber accents used in its
// dashboard blocks, instead of Ant Design's default vivid green/gold/red.
const PASS_COLOR = "#10b981";
const WARN_COLOR = "#f59e0b";
const FAIL_COLOR = "#ef4444";
const WORD_COLOR = "#ef4444";
const NEUTRAL_COLOR = "#64748b";

const STATUS_META = {
  pass: {color: PASS_COLOR, icon: <CheckCircleFilled />, tag: PASS_COLOR, label: () => i18next.t("video:Pass")},
  warn: {color: WARN_COLOR, icon: <ExclamationCircleFilled />, tag: WARN_COLOR, label: () => i18next.t("store:Warning")},
  fail: {color: FAIL_COLOR, icon: <CloseCircleFilled />, tag: FAIL_COLOR, label: () => i18next.t("video:Fail")},
};

const SEVERITY_META = {
  high: {color: FAIL_COLOR, label: () => i18next.t("store:High")},
  medium: {color: WARN_COLOR, label: () => i18next.t("figure:Medium")},
  low: {color: NEUTRAL_COLOR, label: () => i18next.t("store:Low")},
};

function scoreColor(score) {
  if (score >= 75) {return PASS_COLOR;}
  if (score >= 60) {return WARN_COLOR;}
  return FAIL_COLOR;
}

// gaugeOption renders the posture score as a clean 240° arc gauge, matching the
// echarts-driven visual language of the sibling Insights sub-tabs.
function gaugeOption(score, color) {
  return {
    series: [{
      type: "gauge",
      startAngle: 220,
      endAngle: -40,
      min: 0,
      max: 100,
      radius: "96%",
      progress: {show: true, width: 12, roundCap: true, itemStyle: {color}},
      axisLine: {lineStyle: {width: 12, color: [[1, "rgba(128,128,128,0.16)"]]}},
      axisTick: {show: false},
      splitLine: {show: false},
      axisLabel: {show: false},
      pointer: {show: false},
      anchor: {show: false},
      title: {show: false},
      detail: {
        valueAnimation: true,
        offsetCenter: [0, 0],
        fontSize: 34,
        fontWeight: 600,
        color,
        formatter: "{value}",
      },
      data: [{value: score}],
    }],
  };
}

function fill(str, map) {
  let out = str;
  Object.keys(map).forEach((k) => {
    out = out.replace(new RegExp(`\\{${k}\\}`, "g"), map[k]);
  });
  return out;
}

// getCheckText maps a backend check (key + status + meta) to localized display
// text. The backend intentionally ships no prose — all copy lives here so it can
// be translated and kept next to the rest of the store-namespace strings.
function getCheckText(check) {
  const meta = check.meta || {};
  const t = (k) => i18next.t(`store:${k}`);
  switch (check.key) {
  case "prompt_secret_scan": {
    const cats = (meta.categories || []).map((c) => t(`sec.secret.${c}`)).join(", ");
    return {
      title: t("Secrets in agent definition"),
      detail: check.status === "fail"
        ? fill(t("Possible secrets detected in the agent's prompt or description: {c}."), {c: cats})
        : t("No hard-coded secrets found in the agent's prompt or description."),
      fix: t("Remove credentials from the prompt/description and inject them at runtime via a provider or environment variable."),
    };
  }
  case "api_key_exposure":
    return {
      title: t("API key exposure"),
      detail: check.status === "fail"
        ? t("The store's external API key appears in a text field readable by clients.")
        : (meta.hasKey ? t("The external API key is set and not exposed in any readable text field.") : t("No external API key is configured.")),
      fix: t("Rotate the external API key and remove it from the prompt, description, and welcome text."),
    };
  case "content_moderation":
    return {
      title: t("Content moderation list"),
      detail: check.status === "warn"
        ? t("No forbidden words are configured, so user input is not filtered.")
        : fill(t("{n} forbidden word(s) configured for input filtering."), {n: meta.wordCount}),
      fix: t("Add forbidden words in the agent settings to block abusive or sensitive input."),
    };
  case "file_upload_policy":
    return {
      title: t("File upload policy"),
      detail: check.status === "warn"
        ? t("This agent is public and file uploads are enabled, allowing anyone to upload files.")
        : (meta.disabled ? t("File uploads are disabled.") : t("File uploads are enabled.")),
      fix: t("Disable file uploads for public agents, or ensure uploaded files are scanned and access-controlled."),
    };
  case "public_exposure":
    return {
      title: t("Public exposure"),
      detail: check.status === "warn"
        ? t("This agent is published and reachable by anyone in the public Hub.")
        : t("This agent is not published publicly."),
      fix: t("Review the prompt, files, and knowledge base for sensitive content before publishing."),
    };
  case "tool_attack_surface":
    return {
      title: t("Tool & capability surface"),
      detail: check.status === "warn"
        ? fill(t("{tools} tool(s), {skills} skill(s){mcp} are enabled, expanding the attack surface."), {tools: meta.toolCount, skills: meta.skillCount, mcp: meta.hasMcp ? t(" and an MCP server") : ""})
        : t("No external tools, skills, or MCP servers are enabled."),
      fix: t("Enable only the tools and skills this agent needs, and review MCP server permissions."),
    };
  case "access_control":
    return {
      title: t("Co-owner access control"),
      detail: check.status === "warn"
        ? fill(t("{n} owners have write access to this agent."), {n: meta.ownerCount})
        : t("Only the primary owner can modify this agent."),
      fix: t("Keep the co-owner list minimal and remove owners who no longer need write access."),
    };
  case "forbidden_word_violations":
    return {
      title: t("Forbidden-word violations"),
      detail: check.status === "fail"
        ? fill(t("{hits} message(s) tripped the forbidden-word filter in this window."), {hits: meta.hits})
        : fill(t("No forbidden-word violations across {n} scanned message(s)."), {n: meta.messagesScanned}),
      fix: t("Review the flagged messages below and consider blocking or warning the users involved."),
    };
  default:
    return {title: check.key, detail: "", fix: ""};
  }
}

class InsightsSecurity extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      loading: true,
      error: null,
      data: null,
    };
  }

  componentDidMount() {
    this.fetch();
  }

  componentDidUpdate(prevProps) {
    if (prevProps.period !== this.props.period || prevProps.refreshTick !== this.props.refreshTick) {
      this.fetch();
    }
  }

  fetch() {
    const {owner, storeName, period, onLoaded} = this.props;
    this.setState({loading: true, error: null});
    AnalysisBackend.getStoreSecurity(owner, storeName, period)
      .then((res) => {
        if (res.status === "ok") {
          this.setState({loading: false, data: res.data});
          if (onLoaded) {onLoaded(res.data.asOf);}
        } else {
          this.setState({loading: false, error: res.msg});
        }
      })
      .catch((err) => this.setState({loading: false, error: err.message || String(err)}));
  }

  renderProportionBar(data) {
    const total = data.passCount + data.warnCount + data.failCount;
    if (total === 0) {return null;}
    const seg = (n, color) => (n > 0 ? (
      <div style={{width: `${(n / total) * 100}%`, background: color}} />
    ) : null);
    return (
      <div style={{display: "flex", height: 6, borderRadius: 3, overflow: "hidden", marginTop: 18, background: "var(--ant-color-fill-tertiary)"}}>
        {seg(data.passCount, PASS_COLOR)}
        {seg(data.warnCount, WARN_COLOR)}
        {seg(data.failCount, FAIL_COLOR)}
      </div>
    );
  }

  renderScoreCard(data) {
    const color = scoreColor(data.score);
    const stats = [
      {label: i18next.t("store:Passed"), value: data.passCount, color: PASS_COLOR, icon: STATUS_META.pass.icon},
      {label: i18next.t("store:Warnings"), value: data.warnCount, color: WARN_COLOR, icon: STATUS_META.warn.icon},
      {label: i18next.t("application:Failed"), value: data.failCount, color: FAIL_COLOR, icon: STATUS_META.fail.icon},
    ];
    return (
      <Card size="small">
        <Row gutter={[16, 16]} align="middle">
          <Col xs={24} sm={9} md={7} style={{textAlign: "center"}}>
            <div style={{height: 160, position: "relative"}}>
              <ReactEcharts option={gaugeOption(data.score, color)} style={{height: "100%", width: "100%"}} notMerge={true} lazyUpdate={true} />
            </div>
            <div style={{marginTop: -8}}>
              <Tag color={color} style={{margin: 0, fontWeight: 600, fontSize: 13, padding: "0 10px", border: "none"}}>
                {i18next.t("store:Security grade")} {data.grade}
              </Tag>
            </div>
            <div style={{marginTop: 8, color: "var(--ant-color-text-secondary)", fontSize: 13}}>
              <SafetyCertificateOutlined /> {i18next.t("store:Security score")}
            </div>
          </Col>
          <Col xs={24} sm={15} md={17}>
            <Row gutter={[16, 16]}>
              {stats.map((s) => (
                <Col key={s.label} xs={8}>
                  <Statistic
                    title={<span style={{color: s.color}}>{s.icon} {s.label}</span>}
                    value={s.value}
                    valueStyle={{color: s.color}}
                  />
                </Col>
              ))}
            </Row>
            {this.renderProportionBar(data)}
            <Text type="secondary" style={{display: "block", marginTop: 16, fontSize: 13}}>
              {i18next.t("store:This report audits the agent's configuration and recent activity. Address failed and warned checks to improve the score.")}
            </Text>
          </Col>
        </Row>
      </Card>
    );
  }

  renderActivityCards(data) {
    const cards = [
      {label: i18next.t("store:Messages scanned"), value: data.messagesScanned, icon: <MessageOutlined />},
      {label: i18next.t("store:Forbidden-word hits"), value: data.forbiddenWordHits, icon: <StopOutlined />, color: data.forbiddenWordHits > 0 ? FAIL_COLOR : undefined},
      {label: i18next.t("store:Error replies"), value: data.errorMessages, icon: <WarningOutlined />, color: data.errorMessages > 0 ? WARN_COLOR : undefined},
      {
        label: i18next.t("store:Visits"),
        value: data.guestVisits + data.authedVisits,
        icon: <TeamOutlined />,
        suffix: (
          <Text type="secondary" style={{fontSize: 12, marginLeft: 6}}>
            {fill(i18next.t("store:{g} guest / {a} signed-in"), {g: data.guestVisits, a: data.authedVisits})}
          </Text>
        ),
      },
    ];
    return (
      <Row gutter={[16, 16]} style={{marginTop: 16}}>
        {cards.map((c) => (
          <Col key={c.label} xs={12} sm={12} lg={6}>
            <Card size="small">
              <Statistic
                title={<span>{c.icon} {c.label}</span>}
                value={c.value}
                valueStyle={c.color ? {color: c.color} : undefined}
                suffix={c.suffix}
              />
            </Card>
          </Col>
        ))}
      </Row>
    );
  }

  renderChecks(data) {
    const checks = data.checks || [];
    return (
      <Card
        size="small"
        title={<span><SafetyCertificateOutlined /> {i18next.t("store:Configuration checks")}</span>}
        style={{marginTop: 16}}
        styles={{body: {padding: "4px 16px"}}}
      >
        {checks.map((check, idx) => {
          const status = STATUS_META[check.status] || STATUS_META.warn;
          const sev = SEVERITY_META[check.severity] || SEVERITY_META.low;
          const text = getCheckText(check);
          const showFix = check.status !== "pass" && text.fix;
          return (
            <div
              key={check.key}
              style={{
                display: "flex",
                gap: 12,
                alignItems: "flex-start",
                padding: "14px 0",
                borderTop: idx === 0 ? "none" : "1px solid var(--ant-color-border-secondary)",
              }}
            >
              <span style={{color: status.color, fontSize: 18, lineHeight: "22px", flexShrink: 0}}>{status.icon}</span>
              <div style={{flex: 1, minWidth: 0}}>
                <div style={{display: "flex", alignItems: "center", justifyContent: "space-between", gap: 8, flexWrap: "wrap"}}>
                  <Text strong style={{fontSize: 14}}>{text.title}</Text>
                  <span style={{display: "flex", gap: 6, alignItems: "center"}}>
                    <Tag color={sev.color} style={{margin: 0}}>{sev.label()}</Tag>
                    <Tag color={status.tag} style={{margin: 0}}>{status.label()}</Tag>
                  </span>
                </div>
                <div style={{marginTop: 4, color: "var(--ant-color-text-secondary)", fontSize: 13, lineHeight: 1.6}}>
                  {text.detail}
                </div>
                {showFix ? (
                  <div
                    style={{
                      marginTop: 8,
                      padding: "6px 12px",
                      background: "var(--ant-color-fill-quaternary)",
                      borderRadius: 6,
                      fontSize: 13,
                      lineHeight: 1.6,
                    }}
                  >
                    <Text type="secondary" strong>{i18next.t("store:Recommendation")}: </Text>
                    <Text type="secondary">{text.fix}</Text>
                  </div>
                ) : null}
              </div>
            </div>
          );
        })}
      </Card>
    );
  }

  renderTopWords(data) {
    if (!data.topForbiddenWords || data.topForbiddenWords.length === 0) {
      return null;
    }
    const max = data.topForbiddenWords[0].count || 1;
    return (
      <Card
        size="small"
        title={<span><StopOutlined /> {i18next.t("store:Top forbidden words")}</span>}
        style={{marginTop: 16}}
      >
        <List
          size="small"
          dataSource={data.topForbiddenWords}
          renderItem={(it) => (
            <List.Item style={{padding: "6px 0"}}>
              <div style={{display: "grid", gridTemplateColumns: "1fr 140px 48px", gap: 12, alignItems: "center", width: "100%"}}>
                <Text ellipsis={{tooltip: it.label}}>{it.label}</Text>
                <div style={{height: 6, borderRadius: 3, background: "var(--ant-color-fill-tertiary)", overflow: "hidden"}}>
                  <div style={{width: `${Math.round((it.count / max) * 100)}%`, height: "100%", background: WORD_COLOR}} />
                </div>
                <Text type="secondary" style={{textAlign: "right"}}>{it.count}</Text>
              </div>
            </List.Item>
          )}
        />
      </Card>
    );
  }

  renderFlagged(data) {
    if (!data.flaggedMessages || data.flaggedMessages.length === 0) {
      return null;
    }
    const columns = [
      {
        title: i18next.t("general:User"),
        dataIndex: "user",
        key: "user",
        width: 170,
        render: (user) => (user ? <UserLabel user={user} account={this.props.account} size="small" /> : <Text type="secondary">—</Text>),
      },
      {
        title: i18next.t("store:Word"),
        dataIndex: "word",
        key: "word",
        width: 130,
        render: (word) => <Tag color={FAIL_COLOR}>{word}</Tag>,
      },
      {
        title: i18next.t("store:Context"),
        dataIndex: "snippet",
        key: "snippet",
        render: (snippet) => <Text style={{fontSize: 13}}>{snippet}</Text>,
      },
      {
        title: i18next.t("general:Created time"),
        dataIndex: "createdTime",
        key: "createdTime",
        width: 180,
        render: (time) => <Text type="secondary" style={{fontSize: 12}}>{time ? new Date(time).toLocaleString() : "—"}</Text>,
      },
    ];
    return (
      <Card
        size="small"
        title={<span><StopOutlined /> {i18next.t("store:Flagged messages")}</span>}
        style={{marginTop: 16}}
      >
        <Table
          rowKey={(r, i) => `${r.chat}-${i}`}
          columns={columns}
          dataSource={data.flaggedMessages}
          size="small"
          pagination={false}
        />
      </Card>
    );
  }

  render() {
    const {loading, error, data} = this.state;
    if (loading && !data) {
      return <div style={{padding: 40, textAlign: "center"}}><Spin /></div>;
    }
    if (error) {
      return <Alert type="error" message={error} />;
    }
    if (!data) {return null;}
    if (!data.checks || data.checks.length === 0) {
      return <Empty description={i18next.t("store:No security data available")} />;
    }

    return (
      <Spin spinning={loading}>
        <div>
          {this.renderScoreCard(data)}
          {this.renderActivityCards(data)}
          {this.renderChecks(data)}
          {this.renderTopWords(data)}
          {this.renderFlagged(data)}
        </div>
      </Spin>
    );
  }
}

export default InsightsSecurity;
