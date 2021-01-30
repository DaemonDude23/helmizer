FROM docker.io/python:3.9-alpine
ARG HELMIZER_WORK_DIR='/usr/src/app/'
WORKDIR ${HELMIZER_WORK_DIR}
COPY LICENSE README.md CHANGELOG.md src/ ./
RUN addgroup -S python -g 1001 && \
  adduser -S python -G python -u 1001
RUN pip install --no-cache-dir -r requirements.txt
RUN chmod 775 helmizer.py && \
  chown -R 1001:1001 ${HELMIZER_WORK_DIR}
USER python
CMD ["python", "helmizer.py"]
