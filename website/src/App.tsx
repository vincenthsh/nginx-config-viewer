import React, { useEffect, useRef, useState } from 'react';
import MonacoEditor, { RefEditorInstance } from '@uiw/react-monacoeditor';
import Header from './Header';
import './App.css';

import 'monaco-editor-nginx';

const App: React.FC = () => {
  const [theme, setTheme] = useState(
    document.documentElement.dataset.colorMode === 'dark' || !document.documentElement.dataset.colorMode
      ? 'vs-dark'
      : 'vs',
  );
  const [content, setContent] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const editor = useRef<RefEditorInstance>(null);
  function resizeHandle(evn: UIEvent) {
    const { target } = evn;
    const width = (target as Window).innerWidth;
    const height = (target as Window).innerHeight;
    if (editor.current && editor.current.editor) {
      editor.current.editor.layout({ width, height: height - 36 });
    }
  }

  // Function to load nginx config from /raw endpoint
  async function loadConfig() {
    try {
      setLoading(true);
      setError('');
      const response = await fetch('/raw', { cache: 'no-store' });
      if (!response.ok) {
        throw new Error(`Failed to load config: ${response.status}`);
      }
      const text = await response.text();
      setContent(text);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load config');
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    if (editor.current && window) {
      window.addEventListener('resize', resizeHandle, false);
    }
    setTheme(document.documentElement.dataset.colorMode === 'dark' ? 'vs-dark' : 'vs');
    document.addEventListener('colorschemechange', (e) => {
      setTheme(e.detail.colorScheme === 'dark' ? 'vs-dark' : 'vs');
    });

    // Initial load
    loadConfig();

    // Set up Server-Sent Events for live reload
    const eventSource = new EventSource('/events');
    eventSource.onmessage = (event) => {
      if (event.data === 'reload') {
        loadConfig();
      }
    };
    eventSource.onerror = (error) => {
      console.warn('SSE connection error:', error);
    };

    return () => {
      window && window.removeEventListener('resize', resizeHandle, false);
      eventSource.close();
    };
  }, []);
  if (loading) {
    return (
      <div className="App">
        <Header />
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: 'calc(100vh - 36px)' }}>
          Loading nginx configuration...
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="App">
        <Header />
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: 'calc(100vh - 36px)', color: 'red' }}>
          Error: {error}
        </div>
      </div>
    );
  }

  return (
    <div className="App">
      <Header />
      <MonacoEditor
        ref={editor}
        theme={theme === 'vs-dark' ? 'nginx-theme-dark' : 'nginx-theme'}
        language="nginx"
        value={content}
        height="calc(100vh - 36px)"
        options={{
          readOnly: true,
          minimap: { enabled: false },
        }}
      />
    </div>
  );
};

export default App;
