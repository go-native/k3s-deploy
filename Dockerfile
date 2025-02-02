# Use the official nginx image as base
FROM nginx:alpine

# Remove default nginx static assets
RUN rm -rf /usr/share/nginx/html/*

# Create simple index.html with Hello World
RUN echo '<html><body><h1>Hello World</h1></body></html>' > /usr/share/nginx/html/index.html

# Nginx runs on port 8080 as specified in your deploy.yml
EXPOSE 8080

# Copy custom nginx conf to change default port from 80 to 8080
RUN echo 'server { \
    listen 8080; \
    server_name localhost; \
    location / { \
    root /usr/share/nginx/html; \
    index index.html; \
    } \
    }' > /etc/nginx/conf.d/default.conf

# Start Nginx
CMD ["nginx", "-g", "daemon off;"]