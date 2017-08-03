FROM gcr.io/google_appengine/python

RUN apt-get update
RUN apt-get install -y curl openssh-client

# Install Go 1.8.
RUN wget https://storage.googleapis.com/golang/go1.8.linux-amd64.tar.gz
RUN tar -xvf go1.8.linux-amd64.tar.gz
RUN mv go /usr/local
ENV PATH "/usr/local/go/bin:${PATH}"

# Install Node.js 8.x.
RUN curl -sL https://deb.nodesource.com/setup_8.x | bash -
RUN apt-get install -y nodejs

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
# 20 minute timeout.
CMD gunicorn -b :$PORT main:app --timeout 1200 --workers 4
