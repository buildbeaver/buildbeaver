FROM ubuntu:20.04
MAINTAINER NA

RUN apt-get update
RUN apt-get install -y ca-certificates

# We expect the BuildBeaver binaries will have been placed in this relative dir by the build process.
ADD output/go/bin/bb-server /usr/bin/
ADD output/go/bin/bb-tools /usr/bin/

ADD docker/bb-server/start-buildbeaver.sh /usr/bin/

RUN chmod +x /usr/bin/start-buildbeaver.sh

EXPOSE 80 443

VOLUME ["/etc/buildbeaver", "/var/lib/buildbeaver"]

ENTRYPOINT ["/usr/bin/start-buildbeaver.sh"]
