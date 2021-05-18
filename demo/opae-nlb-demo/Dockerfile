FROM fedora:32 as builder

# Fetch dependencies and source code
ARG OPAE_RELEASE=2.0.2-1

RUN dnf install -y git-core wget cmake gcc gcc-c++ make spdlog-devel kernel-headers libedit-devel libuuid-devel json-c-devel cli11-devel python-devel

RUN mkdir -p /usr/src/opae && \
    cd /usr/src/opae && \
    wget -q https://github.com/OPAE/opae-sdk/archive/${OPAE_RELEASE}.tar.gz -O- | tar -zx

# Build OPAE
RUN cd /usr/src/opae/opae-sdk-${OPAE_RELEASE} && \
    mkdir build && \
    cd build && \
    CFLAGS="$CFLAGS -Wno-misleading-indentation" \
    cmake .. && \
    make -j nlb0 nlb3

FROM debian:unstable-20210408-slim
COPY --from=builder /usr/src/opae/opae-sdk-*/build/bin/nlb* /usr/local/bin/
COPY --from=builder /usr/src/opae/opae-sdk-*/build/lib /usr/local/lib/
RUN rm -rf /usr/local/lib/python3
RUN apt-get update && apt-get install -y libjson-c4

COPY test_fpga.sh /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/test_fpga.sh"]
