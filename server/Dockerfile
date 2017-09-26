FROM ubuntu:16.04
# Default is ASCII, but Discovery documents are UTF-8.
ENV LANG C.UTF-8

RUN apt-get update
RUN apt-get install -y curl git openssh-client

# Install Go 1.8.
RUN wget https://storage.googleapis.com/golang/go1.8.linux-amd64.tar.gz
RUN tar -xvf go1.8.linux-amd64.tar.gz
RUN mv go /usr/local
ENV PATH /usr/local/go/bin:$PATH

# Install Node.js 7.x.
# We're using 7.x because there's a doc generation bug in
# google-api-nodejs-client on the latest Node.js version (8.5).
RUN curl -sL https://deb.nodesource.com/setup_7.x | bash -
RUN apt-get install -y nodejs

# Install PHP 7 and Composer.
RUN apt-get install -y php7.0 php7.0-xml
RUN curl https://getcomposer.org/download/1.5.2/composer.phar \
    -o /usr/local/bin/composer
RUN chmod +x /usr/local/bin/composer

# Install pip and setup /env.
RUN apt-get install -y python-pip
RUN pip install virtualenv
RUN virtualenv /env -p python3

# Install Ruby 2.3 and Bundler.
RUN apt-get install -y ruby ruby-dev
RUN gem install bundler --no-ri --no-rdoc

# Set virtualenv environment variables. This is equivalent to running
# source /env/bin/activate
ENV VIRTUAL_ENV /env
ENV PATH /env/bin:$PATH
ADD . /app
WORKDIR /app
RUN pip install -r requirements.txt

# 20 minute timeout so the process is not killed before any task completes.
CMD scripts/git-cookie-authdaemon && \
    gunicorn -b :$PORT main:app --timeout 1200 --workers 4
