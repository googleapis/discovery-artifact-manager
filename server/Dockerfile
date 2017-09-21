FROM gcr.io/google_appengine/python

RUN apt-get update
RUN apt-get install -y curl openssh-client

# Install Go 1.8.
RUN wget https://storage.googleapis.com/golang/go1.8.linux-amd64.tar.gz
RUN tar -xvf go1.8.linux-amd64.tar.gz
RUN mv go /usr/local
ENV PATH /usr/local/go/bin:$PATH

# Install Node.js 8.x.
RUN curl -sL https://deb.nodesource.com/setup_8.x | bash -
RUN apt-get install -y nodejs

# Install PHP 5 and Composer.
RUN apt-get -y install php5
RUN php -r "copy('https://getcomposer.org/installer', 'composer-setup.php');"
RUN php -r "if (hash_file('SHA384', 'composer-setup.php') === '669656bab3166a7aff8a7506b8cb2d1c292f042046c5a994c43155c0be6190fa0355160742ab2e1c88d40d5be660b410') { echo 'Installer verified'; } else { echo 'Installer corrupt'; unlink('composer-setup.php'); } echo PHP_EOL;"
RUN php composer-setup.php --filename=composer --install-dir=/usr/local/bin
RUN php -r "unlink('composer-setup.php');"

# Install Python 3.6.
RUN virtualenv /env -p python3.6

# Install Ruby 2.4.1 and Bundler.
RUN wget https://cache.ruby-lang.org/pub/ruby/2.4/ruby-2.4.1.tar.gz
RUN tar -xvf ruby-2.4.1.tar.gz
RUN cd ruby-2.4.1 && ./configure && make && make install
RUN gem install bundler --no-ri --no-rdoc

# Set virtualenv environment variables. This is equivalent to running
# source /env/bin/activate
ENV VIRTUAL_ENV /env
ENV PATH /env/bin:$PATH
ADD requirements.txt /app/
RUN pip install -r requirements.txt
ADD . /app/

# 20 minute timeout so the process is not killed before any task completes.
CMD scripts/git-cookie-authdaemon && \
    gunicorn -b :$PORT main:app --timeout 1200 --workers 4
