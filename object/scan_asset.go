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

package object

import (
	"fmt"
	"os"

	"github.com/the-open-agent/openagent/scan"
	"github.com/the-open-agent/openagent/util"
)

// ScanResult represents the result of a scan operation
type ScanResult struct {
	RawResult     string `json:"rawResult"`
	Result        string `json:"result"`
	ResultSummary string `json:"resultSummary"`
	Runner        string `json:"runner"`
}

// ScanAsset performs a scan on a target
// @param provider: The provider ID (owner/name) for scan provider
// @param scan: Optional scan ID (owner/name) for saving results to existing scan
// @param targetMode: "Manual Input"
// @param target: IP address or network range
// @param command: Scan command with optional %s placeholder for target
// @param saveToScan: Whether to save results to scan object (true for scan edit page, false for provider edit page)
func ScanAsset(provider, scanParam, targetMode, target, asset, command string, saveToScan bool, lang string) (*ScanResult, error) {
	if saveToScan && scanParam != "" {
		scanObj, err := GetScan(scanParam)
		if err != nil {
			return nil, err
		}
		if scanObj == nil {
			return nil, fmt.Errorf("scan not found")
		}

		scanObj.State = "Pending"
		scanObj.UpdatedTime = util.GetCurrentTime()
		scanObj.Runner = ""
		scanObj.ErrorText = ""
		scanObj.RawResult = ""
		scanObj.Result = ""
		scanObj.ResultSummary = ""
		_, err = UpdateScan(scanParam, scanObj)
		if err != nil {
			return nil, err
		}

		return &ScanResult{
			RawResult: "",
			Result:    "",
		}, nil
	}

	owner := "admin"
	if provider != "" {
		providerObj, err := GetProvider(provider)
		if err == nil && providerObj != nil {
			owner = providerObj.Owner
		}
	}
	return executeScan(provider, scanParam, target, command, owner, lang)
}

func executeScan(provider, scanParam, target, command, owner string, lang string) (*ScanResult, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("error getting hostname: %v", err)
	}

	providerObj, err := GetProvider(provider)
	if err != nil {
		return nil, err
	}
	if providerObj == nil {
		return nil, fmt.Errorf("provider not found")
	}

	scanProvider, err := scan.GetScanProvider(providerObj.Type, providerObj.ClientId, lang)
	if err != nil {
		return nil, err
	}
	if scanProvider == nil {
		return nil, fmt.Errorf("scan provider not supported")
	}

	rawResult, err := scanProvider.Scan(target, command)
	if err != nil {
		return nil, err
	}

	result, err := scanProvider.ParseResult(rawResult)
	if err != nil {
		return nil, err
	}

	resultSummary := scanProvider.GetResultSummary(result)

	return &ScanResult{RawResult: rawResult, Result: result, ResultSummary: resultSummary, Runner: hostname}, nil
}

// GetPendingScans returns all scans with state "Pending"
func GetPendingScans() ([]*Scan, error) {
	scans := []*Scan{}
	err := adapter.engine.Where("state = ?", "Pending").Find(&scans)
	if err != nil {
		return nil, err
	}
	return scans, nil
}

// AtomicClaimScan atomically updates a scan's state from "Pending" to "Running"
func AtomicClaimScan(owner, name, hostname string) (int64, error) {
	affected, err := adapter.engine.Table(&Scan{}).
		Where("owner = ? AND name = ? AND state = ?", owner, name, "Pending").
		Update(map[string]interface{}{
			"state":        "Running",
			"runner":       hostname,
			"updated_time": util.GetCurrentTime(),
		})
	return affected, err
}
