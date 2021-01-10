FROM golang:1.15-buster
LABEL author="Thomas Bellembois"

#
# Build prepare.
#

# TODO: fixme workaround WASM module incompatible Go module
RUN mkdir -p /home/thbellem/workspace
RUN ln -s /go /home/thbellem/workspace/workspace_go

# Creating DB volume directory.
RUN mkdir /data && chown www-data /data

# Creating www directory.
RUN mkdir /var/www-data && chown www-data /var/www-data

# Installing Jade and Rice commands.
RUN go get -v github.com/Joker/jade/cmd/jade
RUN go get -v github.com/GeertJohan/go.rice/rice

#
# Sources.
#

# Getting wasm module sources.
WORKDIR /go/src/github.com/tbellembois/
RUN git clone -b devel https://github.com/tbellembois/gochimitheque-wasm.git

# Copying Chimithèque sources.
WORKDIR /go/src/github.com/tbellembois/gochimitheque/
COPY . .

#
# Build.
#

# Building wasm module.
WORKDIR /go/src/github.com/tbellembois/gochimitheque-wasm
RUN go get -v -d ./...
RUN GOOS=js GOARCH=wasm go build -o wasm .

# Copying WASM module into sources.
RUN cp /go/src/github.com/tbellembois/gochimitheque-wasm/wasm /go/src/github.com/tbellembois/gochimitheque/wasm/

# Installing Chimithèque dependencies.
WORKDIR /go/src/github.com/tbellembois/gochimitheque/
RUN go get -v -d ./...

# Generating code.
RUN go generate

# Building Chimithèque.
RUN go build .

#
# Install.
#

# Installing Chimithèque.
RUN cp /go/src/github.com/tbellembois/gochimitheque/gochimitheque /var/www-data/ \
    && chown www-data /var/www-data/gochimitheque \
    && chmod +x /var/www-data/gochimitheque

#
# Final work.
#

# Cleaning up sources.
RUN rm -Rf /go/src/*

# Copying entrypoint.
COPY docker/entrypoint.sh /
RUN chmod +x /entrypoint.sh

# Container configuration.
USER www-data
WORKDIR /var/www-data
ENTRYPOINT [ "/entrypoint.sh" ]
VOLUME ["/data"]
EXPOSE 8081