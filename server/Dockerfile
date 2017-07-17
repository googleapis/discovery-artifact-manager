FROM gcr.io/google_appengine/python
RUN virtualenv /env -p python3.5

RUN apt-get update
RUN apt-get -y install curl openssh-client

# Install Go 1.8
RUN wget https://storage.googleapis.com/golang/go1.8.linux-amd64.tar.gz
RUN tar -xvf go1.8.linux-amd64.tar.gz
RUN mv go /usr/local
ENV PATH "/usr/local/go/bin:${PATH}"

# Install RVM, Ruby, and Bundler.
RUN gpg --keyserver hkp://keys.gnupg.net --recv-keys 409B6B1796C275462A1703113804BB82D39DC0E3 7D2BAF1CF37B13E2069D6956105BD0E739499BDB
RUN \curl -sSL https://get.rvm.io | bash -s stable --ruby
RUN rvm install 2.4
RUN rvm use 2.4
RUN gem install bundler

# Set virtualenv environment variables. This is equivalent to running
# source /env/bin/activate
ENV VIRTUAL_ENV /env
ENV PATH /env/bin:$PATH
ADD requirements.txt /app/
RUN pip install -r requirements.txt
ADD . /app/
# 10 minute timeout.
CMD gunicorn -b :$PORT main:app --timeout 600 --workers 4
