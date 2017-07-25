FROM gcr.io/google_appengine/python
RUN virtualenv /env -p python3.5

RUN apt-get update
RUN apt-get -y install curl openssh-client

# Install Go 1.8
RUN wget https://storage.googleapis.com/golang/go1.8.linux-amd64.tar.gz
RUN tar -xvf go1.8.linux-amd64.tar.gz
RUN mv go /usr/local
ENV PATH "/usr/local/go/bin:${PATH}"

# Install Ruby and Bundler.
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
# 10 minute timeout.
CMD gunicorn -b :$PORT main:app --timeout 600 --workers 4
