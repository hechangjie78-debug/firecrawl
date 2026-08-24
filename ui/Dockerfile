# syntax=docker/dockerfile:1
FROM node:22-alpine AS build
WORKDIR /app

COPY ingestion-ui/package.json ./
RUN npm install

COPY ingestion-ui/ .
RUN npm run build || npx vite build

FROM nginx:alpine AS runtime
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
