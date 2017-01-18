#!/bin/bash
# Copyright (c) 2016 Alces Software Ltd.
#
# Adapted from https://github.com/paulhammond/jp/blob/master/pkg/package.sh
#
# Licensed under the MIT License. Original license reproduced below.
#
# Copyright (c) 2013-2014 Paul Hammond
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.

# To get this script working you need go set up go to do cross compilation.
#  . for mac/homebrew, run "brew install go --cross-compile-common"
#  . on linux install from source then run something like:
#     GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 ./make.bash --no-clean
#     GOOS=linux GOARCH=386 CGO_ENABLED=0 ./make.bash --no-clean

if [ -z "$2" ]; then
    echo "Usage: $0 <version> <flight release>"
    exit 1
fi

cd $(dirname "$0")/..

go get ./...

pkg="github.com/alces-software/flight-attendant/attendant"
ldflags="-X $pkg.ReleaseDate=$(date +%Y-%m-%d)"
version=${1:-0.0.0}
ldflags="$ldflags -X $pkg.Version=$version"
release=${2:-dev}
ldflags="$ldflags -X $pkg.FlightRelease=$release"

echo "Building for v${version}, release ${release}..."
for DEST in linux-386 linux-amd64 darwin-amd64; do
    OS=${DEST%-*}
    ARCH=${DEST#*-}
    DIR=pkg/$DEST
    mkdir -p $DIR
    echo "Building $OS/$ARCH..."
    GOOS=$OS GOARCH=$ARCH go build \
        -ldflags "${ldflags}" \
        -o $DIR/fly .
done
