require 'formula'

class Fly < Formula
  homepage 'http://alces-flight.com/'
  url 'https://s3-eu-west-1.amazonaws.com/alces-flight/FlightAttendant/darwin-x86_64/fly'
  version '0.1'
  sha256 'c25bb93607e9d1790e8621c5edf89c98d0845cded97b16ca9a9781539da943'

  def install
    bin.install 'fly'
  end
end
