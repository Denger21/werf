---
title: Using mounts
sidebar: how_to
permalink: how_to/mounts.html
author: Artem Kladov <artem.kladov@flant.com>
---

## Task Overview

In this article, we will build an example GO application. Then we will optimize the build instructions to substantial reduce image size with using mount directives.

## Requirements

* Installed [multiwerf](https://github.com/flant/multiwerf) on the host system.

### Select werf version

This command should be run prior running any werf command in your shell session:

```
source <(multiwerf use 1.0)
```

## Sample application

The example application is the [Hotel Booking Example](https://github.com/revel/examples/tree/master/booking), written in [GO](https://golang.org/) for [Revel Framework](https://github.com/revel).

### Building

Create a `booking` directory and place the following `werf.yaml` in the `booking` directory:
{% raw %}
```yaml
project: hotel-booking
---

{{ $_ := set . "GoDlPath" "https://dl.google.com/go/" }}
{{ $_ := set . "GoTarball" "go1.11.1.linux-amd64.tar.gz" }}
{{ $_ := set . "GoTarballChecksum" "sha256:2871270d8ff0c8c69f161aaae42f9f28739855ff5c5204752a8d92a1c9f63993" }}
{{ $_ := set . "BaseImage" "ubuntu:18.04" }}

image: go-booking
from: {{ .BaseImage }}
ansible:
  beforeInstall:
  - name: Install essential utils
    apt:
      name: "{{`{{ item }}`}}"
      update_cache: yes
    with_items:
    - curl
    - git
    - tree
  - name: Download the Go tarball
    get_url:
      url: {{ .GoDlPath }}{{ .GoTarball }}
      dest: /usr/local/src/{{ .GoTarball }}
      checksum:  {{ .GoTarballChecksum }}
  - name: Extract the Go tarball if Go is not yet installed or not the desired version
    unarchive:
      src: /usr/local/src/{{ .GoTarball }}
      dest: /usr/local
      copy: no
  - name: Install additional packages
    apt:
      name: "{{`{{ item }}`}}"
      update_cache: yes
    with_items:
    - gcc
    - sqlite3
    - libsqlite3-dev
  install:
  - name: Getting packages
    shell: |
{{ include "export golang vars" . | indent 6 }}
      go get -v github.com/revel/revel
      go get -v github.com/revel/cmd/revel
      (go get -v github.com/revel/examples/booking/... ; true )
  setup:
  - name: Preparing config and building application
    shell: |
{{ include "export golang vars" . | indent 6 }}
      sed -i 's/^http.addr=$/http.addr=0.0.0.0/' $GOPATH/src/github.com/revel/examples/booking/conf/app.conf
      revel build --run-mode dev github.com/revel/examples/booking /app

# GO-template for exporting environment variables
{{- define "export golang vars" -}}
export GOPATH=/go
export PATH=$GOPATH/bin:$PATH:/usr/local/go/bin
{{- end -}}
```
{% endraw %}

Build the application by executing the following command in the `booking` directory:
```bash
werf build
```

### Running

Run the application by executing the following command in the `booking` directory:
```bash
werf tag --repo booking --tag v1.0
docker run -p 9000:9000 --rm -d booking/go-booking:v1.0 /app/run.sh
```

Check that container is running by executing the following command:
```bash
docker ps
```

You should see a running container with a random name, like this:
```bash
CONTAINER ID  IMAGE         COMMAND        CREATED        STATUS        PORTS                   NAMES
41d6f49798a8  14e6b9c6b93b  "/app/run.sh"  3 minutes ago  Up 3 minutes  0.0.0.0:9000->9000/tcp  infallible_bell
```

Open in a web browser the following URL — [http://localhost:9000](http://localhost:9000).

The `revel framework booking demo` page should open, and you can login by entering `demo/demo` as a login/password.

### Getting the application image size

After tagging we get an image `booking/go-booking:v1.0` according to werf naming rules (read more about naming [here]({{ site.baseurl }}/reference/registry/image_naming.html)).

Determine the image size by executing:

```bash
docker images booking/go-booking:v1.0
```

The output will be something like this:
```bash
REPOSITORY           TAG           IMAGE ID            CREATED             SIZE
booking/go-booking   v1.0          0bf71cb34076        10 minutes ago      1.04 GB
```

You can check the size of all ancestor images. To find ancestor images tags look at the output of the `werf build` command — in the lines like `# Calculated signature c05db314b209a96bd906b77c910d6a5ae76e25f6422bf57f2da37e935805ddca for image/go-booking stage/setup`. The long HEX value is the image tag. E.e. you could see in the output of the `docker images` command like this (TAGs values was cut to fit the web page):

```bash
REPOSITORY            TAG                  IMAGE ID            CREATED             SIZE
image-stage-booking     c05db314b20...ddca   14e6b9c6b93b        21 minutes ago      1.04 GB
image-stage-booking     46fb00c9dda...3ef1   9a34966e6c85        22 minutes ago      938 MB
image-stage-booking     bf057acfb01...5d4b   97ea9a99ddb2        49 minutes ago      805 MB
image-stage-booking     41772c141b1...9a11   66ce7d681e8d        52 minutes ago      84.1 MB
```

Pay attention, that the image size of the application is **above 1 GB**.

## Optimizing

There are often a lot of useless files in the image. In our example application, these are — APT cache and GO sources. Also, after building the application, the GO itself is not needed to run the application and can be removed from the image.

### Optimizing APT cache

[APT](https://wiki.debian.org/Apt) saves the package list in the `/var/lib/apt/lists/` directory and also saves packages in the `/var/cache/apt/` directory when installs them. So, it is useful to store `/var/cache/apt/` outside the image and share it between builds. The `/var/lib/apt/lists/` directory depends on the status of the installed packages, and it's no good to share it between builds, but it is useful to store it outside the image to reduce its size.

To optimize using APT cache add the following directives to `go-booking` image in the config:

```yaml
mount:
- from: tmp_dir
  to: /var/lib/apt/lists
- from: build_dir
  to: /var/cache/apt
```

Read more about mount directives [here]({{ site.baseurl }}/reference/build/mount_directive.html).

The `/var/lib/apt/lists` directory is filling in the build-time, but in the image, it is empty.

The `/var/cache/apt/` directory is caching in the `~/.werf/builds/booking/mount` directory but in the image, it is empty. Mounts work only during werf assembly process. So, if you change stages instructions and rebuild your project, the `/var/cache/apt/` will already contain packages downloaded earlier.

Official Ubuntu image contains special hooks that remove APT cache after image build. To disable these hooks, add the following task to a beforeInstall stage of the config:

```yaml
ansible:
  beforeInstall:
  - name: Disable docker hook for apt-cache deletion
    shell: |
      set -e
      sed -i -e "s/DPkg::Post-Invoke.*//" /etc/apt/apt.conf.d/docker-clean
      sed -i -e "s/APT::Update::Post-Invoke.*//" /etc/apt/apt.conf.d/docker-clean
```

### Optimizing builds

In the example application, the GO is downloaded and extracted. The GO source is not needed in the image. After the application is built, the GO itself is also not needed in the image. So mount `/usr/local/src` and `/usr/local/go` directories to place them outside the image.

Building application on the setup stage uses the `/go` directory, specified in the `GOPATH` environment variable. This directory contains necessary packages and application source. After the build, the result is placed in the `/app` directory, and the `/go` directory is not needed to run the application. So, the `/go` directory can be mounted to a temporary place, outside of the image.

Add the following to mount directives into config:

```yaml
- from: tmp_dir
  to: /go
- from: build_dir
  to: /usr/local/src
- from: build_dir
  to: /usr/local/go
```

### Complete `werf.yaml` config

{% raw %}
```yaml
project: hotel-booking
---

{{ $_ := set . "GoDlPath" "https://dl.google.com/go/" }}
{{ $_ := set . "GoTarball" "go1.11.1.linux-amd64.tar.gz" }}
{{ $_ := set . "GoTarballChecksum" "sha256:2871270d8ff0c8c69f161aaae42f9f28739855ff5c5204752a8d92a1c9f63993" }}
{{ $_ := set . "BaseImage" "ubuntu:18.04" }}

image: go-booking
from: {{ .BaseImage }}
mount:
- from: tmp_dir
  to: /var/lib/apt/lists
- from: build_dir
  to: /var/cache/apt
- from: tmp_dir
  to: /go
- from: build_dir
  to: /usr/local/src
- from: build_dir
  to: /usr/local/go
ansible:
  beforeInstall:
  - name: Disable docker hook for apt cache deletion
    shell: |
      set -e
      sed -i -e "s/DPkg::Post-Invoke.*//" /etc/apt/apt.conf.d/docker-clean
      sed -i -e "s/APT::Update::Post-Invoke.*//" /etc/apt/apt.conf.d/docker-clean
  - name: Install essential utils
    apt:
      name: "{{`{{ item }}`}}"
      update_cache: yes
    with_items:
    - curl
    - git
    - tree
  - name: Download the Go tarball
    get_url:
      url: {{ .GoDlPath }}{{ .GoTarball }}
      dest: /usr/local/src/{{ .GoTarball }}
      checksum:  {{ .GoTarballChecksum }}
  - name: Extract the Go tarball if Go is not yet installed or not the desired version
    unarchive:
      src: /usr/local/src/{{ .GoTarball }}
      dest: /usr/local
      copy: no
  - name: Install additional packages
    apt:
      name: "{{`{{ item }}`}}"
      update_cache: yes
    with_items:
    - gcc
    - sqlite3
    - libsqlite3-dev
  install:
  - name: Getting packages
    shell: |
{{ include "export golang vars" . | indent 6 }}
      go get -v github.com/revel/revel
      go get -v github.com/revel/cmd/revel
      (go get -v github.com/revel/examples/booking/... ; true )
  setup:
  - name: Preparing config and building application
    shell: |
{{ include "export golang vars" . | indent 6 }}
      sed -i 's/^http.addr=$/http.addr=0.0.0.0/' $GOPATH/src/github.com/revel/examples/booking/conf/app.conf
      revel build --run-mode dev github.com/revel/examples/booking /app

# GO-template for exporting environment variables
{{- define "export golang vars" -}}
export GOPATH=/go
export PATH=$GOPATH/bin:$PATH:/usr/local/go/bin
{{- end -}}
```
{% endraw %}

Build the application with the modified config:
```bash
werf build
```

### Running

Before running the modified application, you need to stop running container. Otherwise, the new container can't bind to 9000 port on localhost. E.g., execute the following command to stop last created container:

```bash
docker stop `docker ps -lq`
```

Run the modified application by executing the following command:
```bash
werf tag --repo booking --tag v2.0
docker run -p 9000:9000 --rm -d booking/go-booking:v2.0 /app/run.sh
```

Check that container is running by executing the following command:
```bash
docker ps
```

You should see a running container with a random name, like this:
```bash
CONTAINER ID  IMAGE         COMMAND        CREATED        STATUS        PORTS                   NAMES
88287022813b  c8277cd4a801  "/app/run.sh"  5 seconds ago  Up 3 seconds  0.0.0.0:9000->9000/tcp  naughty_dubinsky
```

Open in a web browser the following URL — [http://localhost:9000](http://localhost:9000).

The `revel framework booking demo` page should open, and you can login by entering `demo/demo` as a login/password.

### Getting images size

Determine the image size of optimized build, by executing:
```bash
docker images booking/go-booking
```

The output will be something like this:
```bash
REPOSITORY            TAG        IMAGE ID         CREATED            SIZE
booking/go-booking    v2.0      0a9943b0da6a     3 minutes ago      335 MB
booking/go-booking    v1.0      0bf71cb34076     15 minutes ago     1.04 GB
```

### Analysis

Werf store build cache for project in the `~/.werf/builds/<project>/` directory. Contents of directories mounted with `from: build_dir` parameter are placed in the `~/.werf/builds/<project>/mount/` directory.

Analyze the structure of the `~/.werf/builds/booking/mount` directory. Execute the following command:

```bash
tree -L 3 ~/.werf/builds/hotel-booking/mount
```

The output will be like this (some lines skipped):
```bash
/home/user/.werf/builds/booking/mount
├── usr-local-go-a179aaae
│   ├── api
│   ├── lib
│   ├── pkg
...
│   └── src
├── usr-local-src-f1bad46a
│   └── go1.11.1.linux-amd64.tar.gz
└── var-cache-apt-28143ccf
    └── archives
        ├── binutils_2.30-21ubuntu1~18.04_amd64.deb
...
        └── xauth_1%3a1.0.10-1_amd64.deb
```

As you may see, there are separate directories on the host for every mount in config exists.

Check the directories size, by executing:
```bash
sudo du -kh --max-depth=1 ~/.werf/builds/hotel-booking/mount
```

The output will be like this:
```bash
49M     /home/user/.werf/builds/booking/mount/var-cache-apt-28143ccf
122M    /home/user/.werf/builds/booking/mount/usr-local-src-f1bad46a
423M    /home/user/.werf/builds/booking/mount/usr-local-go-a179aaae
592M    /home/user/.werf/builds/booking/mount
```

`592MB` is a size of files excluded from image, but these files are accessible, in case of rebuild image and also they can be mounted in other images in this project. E.g., if you add image based on Ubuntu, you can mount `/var/cache/apt` with `from: build_dir` and use already downloaded packages.

Also, approximately `77MB` of space occupy files in directories mounted with `from: tmp_dir`. These files also excluded from the image and deleted from the host at the end of image building.

The total size difference between `v1.0` and `v2.0` images is about 730 MB (the result of 1.04 GB — 335 MB).

**Our example shows that with using werf mounts the image size smaller by more than 68% than the original image size!**

## What Can Be Improved

* Use a smaller base image instead of ubuntu, such as [alpine](https://hub.docker.com/_/alpine/) or [golang](https://hub.docker.com/_/golang/).
* Using [werf artifacts]({{ site.baseurl }}/reference/build/artifact.html) in many cases can give more efficient.
  The size of `/app` directory in the image is about only 17 MB (you can check it by executing `docker run --rm booking/go-booking:v2.0 du -kh --max-depth=0 /app`). So you can build files into the `/app` in werf artifact and then import only the resulting `/app` directory.
