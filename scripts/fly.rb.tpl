require 'formula'

class Fly < Formula
  homepage 'http://alces-flight.com/'
  url 'https://s3-eu-west-1.amazonaws.com/alces-flight/FlightAttendant/darwin-x86_64/fly'
  version '%VERSION%'
  sha256 '%SHA256SUM%'

  def install
    bin.install 'fly'
  end
end
