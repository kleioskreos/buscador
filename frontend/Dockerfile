# ============================================================
# BTR Frontend — React + Vite (build) servido por Nginx
# ============================================================

# ---------- Stage 1: Build con Node ----------
FROM node:22-alpine AS builder

WORKDIR /app

# Dependencias primero (mejor cache de capas)
COPY package.json ./
RUN npm install --no-audit --no-fund

# Codigo fuente
COPY . .

# Build de produccion
RUN npm run build

# ---------- Stage 2: Nginx ----------
FROM nginx:1.27-alpine

# Configuracion personalizada (proxy /api -> backend)
RUN rm -f /etc/nginx/conf.d/default.conf
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Assets compilados
COPY --from=builder /app/dist /usr/share/nginx/html

# Permisos para usuario no-root de nginx
RUN chown -R nginx:nginx /usr/share/nginx/html \
    && chown -R nginx:nginx /var/cache/nginx \
    && touch /var/run/nginx.pid \
    && chown nginx:nginx /var/run/nginx.pid

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost/health || exit 1

STOPSIGNAL SIGQUIT

CMD ["nginx", "-g", "daemon off;"]