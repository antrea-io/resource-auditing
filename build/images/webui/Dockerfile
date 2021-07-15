FROM ubuntu:20.04
RUN apt-get update && apt-get install -y git python3 wget
RUN wget -O - https://raw.githubusercontent.com/alberthier/git-webui/master/install/installer.sh | sh
RUN ln -s /usr/bin/python3 /usr/bin/python
WORKDIR /data/resource-auditing-repo
CMD ["git", "webui"]

