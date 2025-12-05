# Multi-stage build for Frontend
FROM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
# npm ci 대신 npm install 사용 (package-lock.json 동기화 문제 해결)
# 또는 package-lock.json이 최신이면 npm ci 사용 가능
RUN npm install

# Copy source code
COPY . .

# Build the application
# 환경 변수 설정 (빌드 시점에 필요)
ARG REACT_APP_API_URL
ENV REACT_APP_API_URL=${REACT_APP_API_URL}

RUN npm run build

# Production stage
FROM nginx:alpine

# Copy built files from builder
COPY --from=builder /app/build /usr/share/nginx/html

# Copy nginx configuration (optional)
# COPY nginx.conf /etc/nginx/conf.d/default.conf

# Expose port
EXPOSE 80

# Start nginx
CMD ["nginx", "-g", "daemon off;"]

