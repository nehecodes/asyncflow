FROM node:alpine AS builder
WORKDIR /app
COPY package.json package-lock.json /app/
RUN npm ci --prefer-offline

FROM node:alpine
RUN apk update && apk add --no-cache curl=8.19.0-r0
WORKDIR /app
COPY --from=builder /app/node_modules /app/node_modules
COPY . /app/
RUN adduser -D app && chown -R app:app /app
USER app
ENV PATH="/app/node_modules/.bin:$PATH"
EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 CMD [ "pgrep", "-x", "node" ]
ENTRYPOINT [ "node", "app.js" ]