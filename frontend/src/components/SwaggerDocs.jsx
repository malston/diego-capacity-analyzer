// ABOUTME: Swagger UI component for API documentation
// ABOUTME: Fetches OpenAPI spec from backend and renders interactive docs

import SwaggerUI from "swagger-ui-react";
import "swagger-ui-react/swagger-ui.css";

const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";

const SwaggerDocs = () => {
  return (
    <div className="min-h-screen bg-white">
      <SwaggerUI url={`${API_URL}/api/v1/openapi.yaml`} />
    </div>
  );
};

export default SwaggerDocs;
