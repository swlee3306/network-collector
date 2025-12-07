import React from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import './App.css';
import Login from './pages/Login';
import Dashboard from './pages/Dashboard';
import Topology from './pages/Topology';
import Metrics from './pages/Metrics';
import InstanceDetail from './pages/InstanceDetail';
import ProjectComparison from './pages/ProjectComparison';
import ResourceList from './pages/ResourceList';
import PrivateRoute from './components/PrivateRoute';

function App() {
  return (
    <Router>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route
          path="/"
          element={
            <PrivateRoute>
              <Dashboard />
            </PrivateRoute>
          }
        />
        <Route
          path="/topology"
          element={
            <PrivateRoute>
              <Topology />
            </PrivateRoute>
          }
        />
        <Route
          path="/metrics"
          element={
            <PrivateRoute>
              <Metrics />
            </PrivateRoute>
          }
        />
        <Route
          path="/instances"
          element={
            <PrivateRoute>
              <ResourceList />
            </PrivateRoute>
          }
        />
        <Route
          path="/instances/:id"
          element={
            <PrivateRoute>
              <InstanceDetail />
            </PrivateRoute>
          }
        />
        <Route
          path="/projects"
          element={
            <PrivateRoute>
              <ResourceList />
            </PrivateRoute>
          }
        />
        <Route
          path="/projects/compare"
          element={
            <PrivateRoute>
              <ProjectComparison />
            </PrivateRoute>
          }
        />
        <Route
          path="/networks"
          element={
            <PrivateRoute>
              <ResourceList />
            </PrivateRoute>
          }
        />
        <Route
          path="/hypervisors"
          element={
            <PrivateRoute>
              <ResourceList />
            </PrivateRoute>
          }
        />
        <Route
          path="/flavors"
          element={
            <PrivateRoute>
              <ResourceList />
            </PrivateRoute>
          }
        />
        <Route
          path="/volumes"
          element={
            <PrivateRoute>
              <ResourceList />
            </PrivateRoute>
          }
        />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Router>
  );
}

export default App;
