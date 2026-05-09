// Copyright 2023 The OpenAgent Authors. All Rights Reserved.
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

package storage

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
)

type AliyunOssStorageProvider struct {
	client   *oss.Client
	bucket   string
	endpoint string
}

func NewAliyunOssStorageProvider(accessKeyId string, accessKeySecret string, region string, bucket string, endpoint string) (*AliyunOssStorageProvider, error) {
	cfg := oss.LoadDefaultConfig().
		WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKeyId, accessKeySecret, "")).
		WithRegion(region).
		WithEndpoint(endpoint)

	client := oss.NewClient(cfg)
	return &AliyunOssStorageProvider{
		client:   client,
		bucket:   bucket,
		endpoint: endpoint,
	}, nil
}

func (p *AliyunOssStorageProvider) getObjectUrl(key string) string {
	endpoint := p.endpoint
	if !strings.HasPrefix(endpoint, "http") {
		endpoint = "https://" + endpoint
	}
	endpoint = strings.TrimRight(endpoint, "/")
	return fmt.Sprintf("%s/%s/%s", endpoint, p.bucket, key)
}

func (p *AliyunOssStorageProvider) ListObjects(prefix string) ([]*Object, error) {
	result, err := p.client.ListObjectsV2(context.TODO(), &oss.ListObjectsV2Request{
		Bucket: oss.Ptr(p.bucket),
		Prefix: oss.Ptr(prefix),
	})
	if err != nil {
		return nil, err
	}

	var objects []*Object
	for _, obj := range result.Contents {
		key := oss.ToString(obj.Key)
		size := obj.Size
		lastModified := ""
		if obj.LastModified != nil {
			lastModified = obj.LastModified.Format(time.RFC3339)
		}
		objects = append(objects, &Object{
			Key:          key,
			LastModified: lastModified,
			Size:         size,
			Url:          p.getObjectUrl(key),
		})
	}
	return objects, nil
}

func (p *AliyunOssStorageProvider) PutObject(user string, parent string, key string, fileBuffer *bytes.Buffer) (string, error) {
	_, err := p.client.PutObject(context.TODO(), &oss.PutObjectRequest{
		Bucket: oss.Ptr(p.bucket),
		Key:    oss.Ptr(key),
		Body:   fileBuffer,
	})
	if err != nil {
		return "", err
	}
	return p.getObjectUrl(key), nil
}

func (p *AliyunOssStorageProvider) DeleteObject(key string) error {
	_, err := p.client.DeleteObject(context.TODO(), &oss.DeleteObjectRequest{
		Bucket: oss.Ptr(p.bucket),
		Key:    oss.Ptr(key),
	})
	return err
}
