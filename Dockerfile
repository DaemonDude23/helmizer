FROM docker.io/python:3.11-alpine
ARG HELMIZER_WORK_DIR='/usr/src/app/'
# ARG HELMIZER_HELM_VERSION='3.5.2'
WORKDIR ${HELMIZER_WORK_DIR}
COPY LICENSE README.md CHANGELOG.md src/ ./
RUN apk update
RUN apk add curl bash
# RUN wget "https://get.helm.sh/helm-v${HELMIZER_HELM_VERSION}-linux-amd64.tar.gz" && \
#   tar -zxvf "helm-v${HELMIZER_HELM_VERSION}-linux-amd64.tar.gz" && \
#   mv linux-amd64/helm /usr/local/bin/helm
RUN addgroup -S python -g 1001 && \
  adduser -S python -G python -u 1001
RUN pip install --no-cache-dir -r requirements.txt
RUN chmod 775 helmizer.py
  # chown -R 1001:1001 ${HELMIZER_WORK_DIR}
USER python
CMD ["python", "helmizer.py"]
