FROM gcr.io/google_appengine/python
RUN virtualenv /env -p python3.5

RUN apt-get update
RUN apt-get -y install openssh-client

# Install Go 1.8
RUN wget https://storage.googleapis.com/golang/go1.8.linux-amd64.tar.gz
RUN tar -xvf go1.8.linux-amd64.tar.gz
RUN mv go /usr/local
ENV PATH="/usr/local/go/bin:${PATH}"

RUN git clone https://github.com/googleapis/discovery-artifact-manager /discovery-artifact-manager
RUN git config --global user.name "Google APIs"
RUN git config --global user.email googleapis-publisher@google.com

# Set virtualenv environment variables. This is equivalent to running
# source /env/bin/activate
ENV VIRTUAL_ENV /env
ENV PATH /env/bin:$PATH
ADD requirements.txt /app/
RUN pip install -r requirements.txt
ADD . /app/
# 10 minute timeout.
CMD gunicorn -b :$PORT main:app --timeout 600
