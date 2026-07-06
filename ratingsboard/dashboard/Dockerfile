FROM node:20-slim AS runtime
WORKDIR /app
ENV NODE_ENV=production
COPY package*.json ./
COPY server.js ./
USER 1000
EXPOSE 3000
CMD ["node", "server.js"]
