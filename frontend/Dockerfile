# Base build
FROM public.ecr.aws/docker/library/node:26.5.0-alpine3.24 AS base

WORKDIR /app

# Copy package files for dependency installation
COPY package.json package-lock.json ./
RUN npm ci --ignore-scripts && \
    npm cache clean --force

COPY index.html vite.config.ts tsconfig.json ./
COPY /public ./public
COPY /src ./src


# Development build
FROM base AS development
CMD [ "npm", "run", "dev" ]


# Production build
FROM base AS build
RUN npm run build

# https://images.chainguard.dev/directory/image/nginx/versions
FROM cgr.dev/chainguard/nginx:latest AS production
USER nginx:nginx

COPY --chown=root:nginx --chmod=050 --from=build /app/dist /usr/share/nginx/html
