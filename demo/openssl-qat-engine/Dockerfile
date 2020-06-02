ARG FINAL_BASE_IMAGE=clearlinux:base

FROM clearlinux:base as builder

ARG QAT_DRIVER_RELEASE="qat1.7.l.4.9.0-00008"
ARG QAT_ENGINE_VERSION="v0.5.44"
ARG QAT_ZIP_VERSION="v1.0.1"
ARG QAT_ENGINE_FLAGS="--disable-multibuff-offload"

RUN swupd bundle-add --skip-diskspace-check devpkg-systemd devpkg-openssl c-basic wget git patch && \
    git clone -b $QAT_ENGINE_VERSION https://github.com/intel/QAT_Engine && \
    git clone https://github.com/intel/ipp-crypto && \
    git clone -b $QAT_ZIP_VERSION https://github.com/intel/QATzip && \
    wget https://01.org/sites/default/files/downloads/$QAT_DRIVER_RELEASE.tar.gz && \
    tar xf *.tar.gz

ADD 0001-QAT-Compression-drop-DC_CNV_EXTENDED_CAPABILITY-chec.patch /
RUN patch -p1 < /0001-QAT-Compression-drop-DC_CNV_EXTENDED_CAPABILITY-chec.patch
RUN sed -i -e 's/cmn_ko$//' -e 's/lac_kernel$//' quickassist/Makefile && \
    sed -i -e 's:hugepages/qat:hugepages:' quickassist/utilities/libusdm_drv/linux/user_space/qae_mem_hugepage_utils.c && \
    KERNEL_SOURCE_ROOT=/tmp ./configure && \
    make quickassist-all adf-ctl-all && \
    install -m 755 build/libqat_s.so /usr/lib/ && \
    install -m 755 build/libusdm_drv_s.so /usr/lib/ && \
    install -m 755 build/adf_ctl /usr/bin/ && \
    cd /ipp-crypto/sources/ippcp/crypto_mb && \
    sed -i -e '/OPENSSL_USE_STATIC_LIBS/d' CMakeLists.txt && \
    cmake . -B"../build" \
    -DOPENSSL_INCLUDE_DIR=/usr/include/openssl \
    -DOPENSSL_LIBRARIES=/usr/lib64 \
    -DOPENSSL_ROOT_DIR=/usr/bin/openssl && \
    cd ../build && \
    make crypto_mb && make install && \
    install -m 755 bin/vfy_ifma_cp_rsa_mb /usr/bin/ && \
    install -m 755 bin/vfy_ifma_rsa_mb /usr/bin/ && \
    cd /QAT_Engine && \
    ./autogen.sh && \
    ./configure \
    $QAT_ENGINE_FLAGS \
    --with-qat_dir=/ \
    --with-openssl_install_dir=/usr/lib64 \
    --with-qat_install_dir=/usr/lib \
    --with-multibuff_dir=/ipp-crypto/sources/ippcp/crypto_mb \
    --with-multibuff_install_dir=/usr/local/lib \
    --enable-openssl_install_build_arch_path && \
    make && make install
ADD 0001-qzip-fix-format-truncation.patch /
RUN cd /QATzip && GIT_COMMITTER_EMAIL=foo@example.com git am /0001-qzip-fix-format-truncation.patch && \
    ./configure --with-ICP_ROOT=/ && make all install

FROM $FINAL_BASE_IMAGE

COPY --from=builder /usr/lib/libqat_s.so /usr/lib/
COPY --from=builder /usr/lib/libusdm_drv_s.so /usr/lib/
COPY --from=builder /usr/local/lib64/libqatzip.a /usr/lib64/
COPY --from=builder /usr/local/include/qatzip.h /usr/include
COPY --from=builder /usr/local/lib64/libqatzip.so /usr/lib64/libqatzip.so
COPY --from=builder /usr/local/lib64/libqatzip.so /usr/lib64/libqatzip.so.1.0.1
COPY --from=builder /usr/local/bin/qzip /usr/bin
COPY --from=builder /usr/bin/adf_ctl /usr/bin
COPY --from=builder /usr/bin/vfy_ifma_cp_rsa_mb /usr/bin
COPY --from=builder /usr/bin/vfy_ifma_rsa_mb /usr/bin
COPY --from=builder /usr/lib64/engines-1.1/qat.so /usr/lib64/engines-1.1/qat.so
COPY --from=builder /LICENSE.GPL /usr/share/package-licenses/libqat/LICENSE.GPL
COPY --from=builder /QAT_Engine/LICENSE /usr/share/package-licenses/QAT_Engine/LICENSE
COPY --from=builder /ipp-crypto/LICENSE /usr/share/package-licenses/ipp-crypto/LICENSE
COPY --from=builder /QATzip/LICENSE /usr/share/package-licenses/QATzip/LICENSE

RUN ldconfig -v
