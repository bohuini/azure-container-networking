ARG ARCH
ARG OS_VERSION
FROM --platform=linux/${ARCH} mcr.microsoft.com/cbl-mariner/base/core:2.0 AS tar
RUN tdnf install -y tar
RUN tdnf install -y unzip
RUN tdnf upgrade -y && tdnf install -y ca-certificates

FROM tar AS azure-vnet
ARG AZCNI_VERSION=v1.6.0
ARG VERSION
ARG OS
ARG ARCH
WORKDIR /azure-container-networking
COPY . .
RUN curl -LO --cacert /etc/ssl/certs/ca-certificates.crt https://dev.azure.com/msazure/_apis/resources/Containers/180623237/output?itemPath=output%2Fbins%2Fazure-vnet-cni-windows-amd64-v1.6.0-46-g171c75b2.zip && unzip -o azure-vnet-cni-windows-amd64-v1.6.0-46-g171c75b2.zip

FROM --platform=linux/${ARCH} mcr.microsoft.com/cbl-mariner/base/core:2.0 AS compressor
ARG OS
WORKDIR /dropgz
COPY dropgz .
COPY --from=azure-vnet /azure-container-networking/cni/azure-$OS-swift-overlay.conflist pkg/embed/fs/azure-swift-overlay.conflist
COPY --from=azure-vnet /azure-container-networking/cni/azure-$OS-swift-overlay-dualstack.conflist pkg/embed/fs/azure-swift-overlay-dualstack.conflist
# COPY --from=azure-vnet /azure-container-networking/azure-vnet.exe pkg/embed/fs <- DELETE
COPY --from=azure-vnet /azure-container-networking/azure-vnet-stateless.exe pkg/embed/fs/azure-vnet.exe
COPY --from=azure-vnet /azure-container-networking/azure-vnet-telemetry.exe pkg/embed/fs
COPY --from=azure-vnet /azure-container-networking/azure-vnet-ipam.exe pkg/embed/fs
COPY --from=azure-vnet /azure-container-networking/azure-vnet-telemetry.config pkg/embed/fs
RUN cd pkg/embed/fs/ && sha256sum * > sum.txt
RUN gzip --verbose --best --recursive pkg/embed/fs && for f in pkg/embed/fs/*.gz; do mv -- "$f" "${f%%.gz}"; done

FROM --platform=linux/${ARCH} mcr.microsoft.com/oss/go/microsoft/golang:1.21 AS dropgz
ARG VERSION
WORKDIR /dropgz
COPY --from=compressor /dropgz .
RUN GOOS=windows CGO_ENABLED=0 go build -a -o bin/dropgz.exe -trimpath -ldflags "-X github.com/Azure/azure-container-networking/dropgz/internal/buildinfo.Version="$VERSION"" -gcflags="-dwarflocationlists=true" main.go

FROM mcr.microsoft.com/windows/nanoserver:${OS_VERSION} as windows
COPY --from=dropgz /dropgz/bin/dropgz.exe dropgz.exe
ENTRYPOINT [ "dropgz.exe" ]