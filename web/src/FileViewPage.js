// Copyright 2025 The OpenAgent Authors. All Rights Reserved.
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
import {Button, Popover, Table} from "antd";
import {LeftOutlined} from "@ant-design/icons";
import i18next from "i18next";
import * as Setting from "./Setting";
import * as FileBackend from "./backend/FileBackend";
import * as VectorBackend from "./backend/VectorBackend";
import Editor from "./common/Editor";

class FileViewPage extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      fileName: decodeURIComponent(props.match.params.fileName),
      file: null,
      vectors: [],
      loading: true,
      pagination: {
        current: 1,
        pageSize: 10,
        total: 0,
      },
    };
  }

  UNSAFE_componentWillMount() {
    this.getFile();
  }

  getFile = () => {
    FileBackend.getFile("admin", this.state.fileName).then((res) => {
      if (res.status === "ok") {
        this.setState({file: res.data});
        this.fetchVectors(res.data, {current: 1, pageSize: 10});
      } else {
        Setting.showMessage("error", `${i18next.t("general:Failed to get")}: ${res.msg}`);
        this.setState({loading: false});
      }
    });
  };

  getObjectKey = (file) => {
    return file.name.startsWith(`${file.store}_`)
      ? file.name.substring(file.store.length + 1)
      : file.name;
  };

  fetchVectors = (file, pagination) => {
    this.setState({loading: true});
    const objectKey = this.getObjectKey(file);
    const {current, pageSize} = pagination;
    VectorBackend.getVectors("admin", file.store, current, pageSize, "file", objectKey, "index", "ascend")
      .then((res) => {
        this.setState({loading: false});
        if (res.status === "ok") {
          this.setState({
            vectors: res.data || [],
            pagination: {
              ...pagination,
              total: res.data2 || 0,
            },
          });
        } else {
          Setting.showMessage("error", `${i18next.t("general:Failed to get")}: ${res.msg}`);
        }
      });
  };

  handleTableChange = (pagination) => {
    this.fetchVectors(this.state.file, pagination);
  };

  renderTable = () => {
    const {vectors, pagination, loading, file} = this.state;

    const columns = [
      {
        title: i18next.t("vector:Index"),
        dataIndex: "index",
        key: "index",
        width: "80px",
      },
      {
        title: i18next.t("general:Text"),
        dataIndex: "text",
        key: "text",
        render: (text) => (
          <Popover placement="left" content={
            <Editor value={text} lang="text" dark readOnly height="300px" width="800px" />
          } title="" trigger="hover">
            <div style={{maxWidth: "500px"}}>
              {Setting.getShortText(text, 100)}
            </div>
          </Popover>
        ),
      },
      {
        title: i18next.t("chat:Token count"),
        dataIndex: "tokenCount",
        key: "tokenCount",
        width: "110px",
      },
    ];

    const paginationProps = {
      current: pagination.current,
      pageSize: pagination.pageSize,
      total: pagination.total,
      showQuickJumper: true,
      showSizeChanger: true,
      pageSizeOptions: ["10", "20", "50", "100"],
      showTotal: () => i18next.t("general:{total} in total").replace("{total}", pagination.total),
    };

    return (
      <div>
        <Table
          scroll={{x: "max-content"}}
          columns={columns}
          dataSource={vectors}
          rowKey="name"
          size="middle"
          bordered
          pagination={paginationProps}
          title={() => (
            <div>
              <Button
                type="text"
                size="small"
                icon={<LeftOutlined />}
                onClick={() => this.props.history.goBack()}
                style={{marginRight: "8px"}}
              />
              {i18next.t("general:Vectors")}
              {file ? ` - ${file.filename}` : ""}
            </div>
          )}
          loading={loading ? {tip: i18next.t("general:Loading")} : false}
          onChange={this.handleTableChange}
        />
      </div>
    );
  };

  render() {
    return (
      <div>
        {this.renderTable()}
      </div>
    );
  }
}

export default FileViewPage;
