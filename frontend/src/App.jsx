import { AuthProvider, useAuth } from "./contexts/AuthContext";
import TASCapacityAnalyzer from "./TASCapacityAnalyzer";
import Login from "./components/Login";
import SwaggerDocs from "./components/SwaggerDocs";
import { Loader } from "lucide-react";
import "./index.css";

// Main app component that handles routing based on auth
const AppContent = () => {
  const { isAuthenticated, loading } = useAuth();

  // Serve API docs without auth (normalize to handle trailing slash)
  const path = window.location.pathname.replace(/\/$/, "");
  if (path === "/docs") {
    return <SwaggerDocs />;
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950 flex items-center justify-center">
        <div className="text-center">
          <Loader className="w-8 h-8 text-blue-400 animate-spin mx-auto mb-4" />
          <p className="text-slate-400 text-sm font-mono">Loading...</p>
        </div>
      </div>
    );
  }

  return isAuthenticated ? <TASCapacityAnalyzer /> : <Login />;
};

function App() {
  return (
    <AuthProvider>
      <AppContent />
    </AuthProvider>
  );
}

export default App;
