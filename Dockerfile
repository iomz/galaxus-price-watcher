FROM ubuntu:xenial-20210429

LABEL maintainer="Iori Mizutani <iori.mizutani@gmail.com>"

RUN apt-get update \
 && apt-get install curl default-jre gcc firefox xvfb -y \
 && apt-get clean \
 && apt-get autoremove -y

RUN curl -LSs https://dl.google.com/go/go1.16.5.linux-amd64.tar.gz -o go.tar.gz \
 && rm -rf /usr/local/go && tar -C /usr/local -xzf go.tar.gz \
 && rm -v go.tar.gz
ENV PATH=${PATH}:/usr/local/go/bin

RUN mkdir -p /build
COPY vendor /build/
COPY go.mod /build/
COPY go.sum /build/
COPY main.go /build/
WORKDIR /build
RUN go mod vendor
RUN go build -mod=vendor -o /usr/local/bin/gpw .

#ENTRYPOINT ["/app/geckodriver", "--version"]
#ENTRYPOINT ["java", "-jar", "/app/selenium-server.jar", "--version"]
#CMD ["/bin/bash"]
ENTRYPOINT ["/usr/local/bin/gpw"]
CMD ["-c", "/app/gpw.toml"]
