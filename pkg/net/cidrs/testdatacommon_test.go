/*
Copyright 2022 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cidrs

import (
	"net/netip"
)

// common test data
var testCIDRS = map[netip.Prefix]string{
	netip.MustParsePrefix("35.180.0.0/16"):    "eu-west-3",
	netip.MustParsePrefix("52.94.76.0/22"):    "us-west-2",
	netip.MustParsePrefix("52.93.127.17/32"):  "eu-west-3",
	netip.MustParsePrefix("52.93.127.172/31"): "eu-west-3",
	netip.MustParsePrefix("52.93.127.173/32"): "us-east-1",
	netip.MustParsePrefix("52.93.127.174/32"): "ap-northeast-1",
	netip.MustParsePrefix("52.93.127.175/32"): "ap-northeast-1",
	netip.MustParsePrefix("52.93.127.176/32"): "ap-northeast-1",
	netip.MustParsePrefix("52.93.127.177/32"): "ap-northeast-1",
	netip.MustParsePrefix("52.93.127.178/32"): "ap-northeast-1",
	netip.MustParsePrefix("52.93.127.179/32"): "ap-northeast-1",
	// ipv6
	netip.MustParsePrefix("2400:6500:0:9::2/128"): "ap-southeast-3",
	netip.MustParsePrefix("2600:1f01:4874::/47"):  "us-west-2",
}

// common test cases
var testCases = []struct {
	Addr           netip.Addr
	ExpectedRegion string
}{
	{Addr: netip.MustParseAddr("35.180.1.1"), ExpectedRegion: "eu-west-3"},
	{Addr: netip.MustParseAddr("35.250.1.1"), ExpectedRegion: ""},
	{Addr: netip.MustParseAddr("35.0.1.1"), ExpectedRegion: ""},
	{Addr: netip.MustParseAddr("52.94.76.1"), ExpectedRegion: "us-west-2"},
	{Addr: netip.MustParseAddr("52.94.77.1"), ExpectedRegion: "us-west-2"},
	{Addr: netip.MustParseAddr("52.93.127.172"), ExpectedRegion: "eu-west-3"},
	// ipv6
	{Addr: netip.MustParseAddr("2400:6500:0:9::2"), ExpectedRegion: "ap-southeast-3"},
	{Addr: netip.MustParseAddr("2400:6500:0:9::1"), ExpectedRegion: ""},
	{Addr: netip.MustParseAddr("2400:6500:0:9::3"), ExpectedRegion: ""},
	{Addr: netip.MustParseAddr("2600:1f01:4874::47"), ExpectedRegion: "us-west-2"},
}
