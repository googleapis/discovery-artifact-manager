FROM ubuntu:16.04
# Default is ASCII, but Discovery documents are UTF-8.
ENV LANG C.UTF-8

RUN apt-get update

# Install the latest stable version of git.
RUN apt-get install -y software-properties-common
RUN add-apt-repository -y ppa:git-core/ppa
RUN apt-get update
RUN apt-get install -y git

RUN apt-get install -y curl openssh-client wget

# Install Go 1.12.
RUN wget https://storage.googleapis.com/golang/go1.12.linux-amd64.tar.gz
RUN tar -xvf go1.12.linux-amd64.tar.gz
RUN mv go /usr/local
ENV PATH /usr/local/go/bin:$PATH

# Install Node.js 8.x.
# We need to use 8.x because generate.ts in google-cloud-nodejs-client
# uses async function and 8.x is LTS release
RUN curl -sL https://deb.nodesource.com/setup_8.x | bash -
RUN apt-get install -y nodejs

# Install PHP 7 and Composer.
RUN apt-get install -y php7.0 php7.0-xml
RUN curl https://getcomposer.org/download/1.5.2/composer.phar \
    -o /usr/local/bin/composer
RUN chmod +x /usr/local/bin/composer

# Install pip and setup /env.
RUN apt-get install -y python3-pip
RUN pip3 install virtualenv
RUN virtualenv /env -p python3

# Install Ruby 2.6.5 and Bundler.
ARG ruby_version=2.6.5
RUN apt-get install -y libssl-dev libreadline-dev zlib1g-dev
RUN mkdir -p /tmp/ruby-build \
    && cd /tmp/ruby-build \
    && git clone https://github.com/rbenv/ruby-build.git \
    && ./ruby-build/bin/ruby-build --keep "${ruby_version}" /usr/local/ruby \
    && rm -rf /tmp/ruby-build
ENV PATH /usr/local/ruby/bin:$PATH
RUN gem install bundler:1.17.3 bundler:2.0.2 --no-document

# Set virtualenv environment variables. This is equivalent to running
# source /env/bin/activate
ENV VIRTUAL_ENV /env
ENV PATH /env/bin:$PATH
ADD . /app
WORKDIR /app
RUN pip install -r requirements.txt

# 1 hour timeout so the process is not killed before any task completes.
CMD scripts/git-cookie-authdaemon && \
    gunicorn -b :$PORT main:app --timeout 3600 --workers 4
