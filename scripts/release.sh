#!/bin/bash
if [ -z "$AWS_ACCESS_KEY_ID" ]; then
    echo "$0: must set AWS_ACCESS_KEY_ID"
    exit 1
fi

if [ -z "$AWS_SECRET_ACCESS_KEY" ]; then
    echo "$0: must set AWS_SECRET_ACCESS_KEY"
    exit 1
fi

if [ "$1" == "--dry-run" ]; then
    dry_run="--dry-run"
    shift
fi

cd $(dirname "$0")/..

sha256sum=$(shasum -a 256 pkg/darwin-amd64/fly | cut -f1 -d " ")
if [ -z "$sha256sum" ]; then
    echo "Unable to determine checksum."
    exit 1
fi
version=$(pkg/darwin-amd64/fly --version | cut -f3 -d" " | cut -c2-)
if [ -z "$version" ]; then
    echo "Unable to determine version."
    exit 1
fi

s3cmd put ${dry_run} -P pkg/darwin-amd64/fly s3://alces-flight/FlightAttendant/$version/darwin-x86_64/fly
s3cmd put ${dry_run} -P pkg/linux-i386/fly s3://alces-flight/FlightAttendant/$version/linux-i386/fly
s3cmd put ${dry_run} -P pkg/linux-amd64/fly s3://alces-flight/FlightAttendant/$version/linux-x86_64/fly
sed -e "s/%SHA256SUM%/$sha256sum/g" -e "s/%VERSION%/$version/g" scripts/fly.rb.tpl > /tmp/fly.rb
s3cmd put ${dry_run} -P /tmp/fly.rb s3://alces-flight/FlightAttendant/fly.rb
#rm -f /tmp/fly.rb
