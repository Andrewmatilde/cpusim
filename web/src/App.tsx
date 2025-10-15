import { BrowserRouter, Routes, Route, Link, useLocation } from "react-router-dom"
import { Dashboard } from "@/components/Dashboard"
import { ExperimentGroups } from "@/components/ExperimentGroups"
import { Activity, Layers } from "lucide-react"
import { Toaster } from "@/components/ui/sonner"

function Navigation() {
  const location = useLocation()

  return (
    <header className="border-b">
      <div className="container mx-auto px-4 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Activity className="h-6 w-6" />
            <h1 className="text-2xl font-bold">CPU Simulation Dashboard v0.8.0</h1>
          </div>
          <nav className="flex gap-4">
            <Link
              to="/"
              className={`flex items-center gap-2 px-4 py-2 rounded-md transition-colors ${
                location.pathname === "/"
                  ? "bg-primary text-primary-foreground"
                  : "hover:bg-accent"
              }`}
            >
              <Activity className="h-4 w-4" />
              Dashboard
            </Link>
            <Link
              to="/groups"
              className={`flex items-center gap-2 px-4 py-2 rounded-md transition-colors ${
                location.pathname === "/groups"
                  ? "bg-primary text-primary-foreground"
                  : "hover:bg-accent"
              }`}
            >
              <Layers className="h-4 w-4" />
              Experiment Groups
            </Link>
          </nav>
        </div>
      </div>
    </header>
  )
}

function App() {
  return (
    <BrowserRouter>
      <div className="min-h-screen bg-background">
        <Toaster />
        <Navigation />

        <main className="container mx-auto px-4 py-6">
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/groups" element={<ExperimentGroups />} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  )
}

export default App