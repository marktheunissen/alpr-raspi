FROM ubuntu:16.04

RUN apt-get update && apt-get install -y \
    motion \
    && rm -rf /var/lib/apt/lists/

ENTRYPOINT ["/usr/bin/motion"]
