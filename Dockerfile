FROM golang:1.16-rc-buster
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

# Installing Jade command.
# TODO: fixme
#RUN go get -v github.com/Joker/jade/cmd/jade
COPY jade /go/bin/

#
# Sources.
#

# Getting wasm module sources.
WORKDIR /go/src/github.com/tbellembois/
# prod.
#RUN git clone -b devel https://github.com/tbellembois/gochimitheque-wasm.git
# devel. with:
# sudo mount --bind ~/workspace/workspace_go/src/github.com/tbellembois/gochimitheque-wasm ./bind-gochimitheque-wasm
COPY ./bind-gochimitheque-wasm ./gochimitheque-wasm

# Copying Chimithèque sources.
WORKDIR /go/src/github.com/tbellembois/gochimitheque/
COPY . .
COPY .git/ ./.git/

#
# Build.
#

# Building wasm module.
WORKDIR /go/src/github.com/tbellembois/gochimitheque-wasm
RUN GOOS=js GOARCH=wasm go get -v -d ./...
RUN GOOS=js GOARCH=wasm go build -o wasm .

# Copying WASM module into sources.
RUN cp /go/src/github.com/tbellembois/gochimitheque-wasm/wasm /go/src/github.com/tbellembois/gochimitheque/wasm/

# Installing Chimithèque dependencies.
WORKDIR /go/src/github.com/tbellembois/gochimitheque/
RUN go get -v -d ./...

# Generating code.
RUN go generate

# Building Chimithèque.
RUN if [ ! -z "$GitCommit" ]; then export GIT_COMMIT=$GitCommit; else export GIT_COMMIT=$(git rev-list -1 HEAD); fi; echo "version=$GIT_COMMIT" ;go build -ldflags "-X main.GitCommit=$GIT_COMMIT"

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