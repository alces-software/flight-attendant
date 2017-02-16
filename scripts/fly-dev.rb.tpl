require 'formula'

class FlyDev < Formula
  homepage 'http://alces-flight.com/'
  url 'https://s3-eu-west-1.amazonaws.com/alces-flight/FlightAttendant/%VERSION%/darwin-x86_64/fly'
  version '%VERSION%'
  sha256 '%SHA256SUM%'

  def install
    bin.install('fly' => 'fly-dev')
  end
end
