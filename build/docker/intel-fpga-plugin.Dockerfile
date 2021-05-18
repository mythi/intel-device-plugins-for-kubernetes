# Copyright 2021 Intel Corporation. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# GOLANG_BASE can be used to make the build reproducible by choosing an
# image by its hash:
# GOLANG_BASE=golang@sha256:9d64369fd3c633df71d7465d67d43f63bb31192193e671742fa1c26ebc3a6210
#
# This is used on release branches before tagging a stable version.
# The main branch defaults to using the latest Golang base image.
FROM golang:1.16 as builder

WORKDIR /intel-device-plugins-for-kubernetes
COPY . .

RUN cd cmd/fpga_plugin; CGO_ENABLED=0 go install; cd -
RUN install -D /go/bin/fpga_plugin /install_root/usr/local/bin/intel_fpga_device_plugin \
    && install -D LICENSE /install_root/usr/local/share/package-licenses/intel-device-plugins-for-kubernetes/LICENSE \
    && scripts/copy-modules-licenses.sh ./cmd/fpga_plugin /install_root/usr/local/share/

FROM gcr.io/distroless/static
COPY --from=builder /install_root /
ENTRYPOINT ["/usr/local/bin/intel_fpga_device_plugin"]
