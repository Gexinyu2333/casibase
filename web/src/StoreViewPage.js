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
import {Avatar, Button, Card, Divider, Spin, Tag, Typography} from "antd";
import {CommentOutlined, FolderOpenOutlined} from "@ant-design/icons";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import * as StoreBackend from "./backend/StoreBackend";
import * as Setting from "./Setting";
import i18next from "i18next";
import FileTree from "./FileTree";
import {getChatUrl} from "./StoreHubDrawer";

const {Text, Title} = Typography;

class StoreViewPage extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      owner: props.match.params.owner,
      storeName: props.match.params.storeName,
      store: null,
      loading: true,
    };
  }

  componentDidMount() {
    this.getStore();
  }

  getStore() {
    StoreBackend.getStore(this.state.owner, this.state.storeName)
      .then((res) => {
        if (res.status === "ok") {
          const store = res.data;
          if (store && typeof res.data2 === "string" && res.data2 !== "") {
            store.error = res.data2;
          }
          this.setState({store, loading: false});
        } else {
          Setting.showMessage("error", `${i18next.t("general:Failed to get")}: ${res.msg}`);
          this.setState({loading: false});
        }
      });
  }

  handleStartChat() {
    const {store} = this.state;
    if (!store) {return;}
    if (store.endpoint) {
      window.open(getChatUrl(store), "_blank", "noopener,noreferrer");
    } else {
      this.props.history.push(`/stores/${store.owner}/${store.name}/chat`);
    }
  }

  renderHeader(store) {
    const initials = (store.displayName || store.name || "?")[0].toUpperCase();
    const authorName = store.author || store.owner;

    return (
      <div style={{display: "flex", alignItems: "flex-start", gap: 20, marginBottom: 20, flexWrap: "wrap"}}>
        {store.avatar ? (
          <Avatar size={80} src={store.avatar} style={{flexShrink: 0}} />
        ) : (
          <Avatar size={80} style={{backgroundColor: Setting.getAvatarColor(store.name), flexShrink: 0, fontSize: 32}}>
            {initials}
          </Avatar>
        )}
        <div style={{flex: 1, minWidth: 0}}>
          <div style={{display: "flex", alignItems: "center", gap: 12, flexWrap: "wrap", marginBottom: 6}}>
            <Title level={3} style={{margin: 0}}>
              {store.displayName || store.name}
            </Title>
            <Button
              type="primary"
              icon={<CommentOutlined />}
              onClick={() => this.handleStartChat()}
            >
              {i18next.t("store:Start Chat")}
            </Button>
          </div>
          <Text type="secondary" style={{fontSize: 14}}>
            {i18next.t("store:By")} <strong>{authorName}</strong>
          </Text>
          {store.affiliation ? (
            <div style={{fontSize: 13, color: "var(--ant-color-text-tertiary)", marginTop: 2}}>
              {store.affiliation}
            </div>
          ) : null}
          <div style={{marginTop: 8, display: "flex", flexWrap: "wrap", gap: 6}}>
            {store.subject ? <Tag color="purple">{store.subject}</Tag> : null}
            {store.grade ? <Tag color="cyan">{store.grade}</Tag> : null}
            {store.topic ? <Tag color="geekblue">{store.topic}</Tag> : null}
          </div>
          {store.brief ? (
            <div style={{marginTop: 8, fontSize: 14, color: "var(--ant-color-text-secondary)", maxWidth: 640}}>
              {store.brief}
            </div>
          ) : null}
        </div>
      </div>
    );
  }

  renderReadme(store) {
    const content = store.description || store.prompt || store.welcomeText || "";
    if (!content) {return null;}

    return (
      <Card
        title={
          <div style={{display: "flex", alignItems: "center", gap: 8}}>
            <FolderOpenOutlined />
            <span>{i18next.t("general:Description")}</span>
          </div>
        }
        style={{marginTop: 16}}
        styles={{body: {padding: "20px 24px"}}}
      >
        <div className="markdown-body" style={{fontSize: 14, lineHeight: "1.6"}}>
          <ReactMarkdown remarkPlugins={[remarkGfm]}>
            {content}
          </ReactMarkdown>
        </div>
      </Card>
    );
  }

  render() {
    const {store, loading} = this.state;

    if (loading) {
      return (
        <div style={{display: "flex", justifyContent: "center", alignItems: "center", height: "calc(100vh - 120px)"}}>
          <Spin size="large" tip={i18next.t("general:Loading...")} />
        </div>
      );
    }

    if (!store) {
      return null;
    }

    return (
      <div style={{padding: "24px 32px", maxWidth: 1200, margin: "0 auto"}}>
        {this.renderHeader(store)}
        <Divider style={{margin: "0 0 16px"}} />
        <FileTree
          account={this.props.account}
          store={store}
          onUpdateStore={(updatedStore) => {
            this.setState({store: updatedStore});
            Setting.submitStoreEdit(updatedStore);
          }}
          onRefresh={() => this.getStore()}
        />
        {this.renderReadme(store)}
      </div>
    );
  }
}

export default StoreViewPage;
