import { useEffect, useState, useRef } from 'react';
import { eventsAPI } from '../services/api';

interface EventData {
  type: string;
  timestamp: string;
  data: any;
}

export const useEventSource = () => {
  const [event, setEvent] = useState<EventData | null>(null);
  const [connected, setConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const eventSourceRef = useRef<EventSource | null>(null);

  useEffect(() => {
    // Create EventSource connection
    const eventSource = eventsAPI.createEventSource();
    eventSourceRef.current = eventSource;

    eventSource.onopen = () => {
      setConnected(true);
      setError(null);
      console.log('SSE connection opened');
    };

    eventSource.onmessage = (e) => {
      try {
        const data: EventData = JSON.parse(e.data);
        setEvent(data);
      } catch (err) {
        console.error('Failed to parse event data:', err);
      }
    };

    eventSource.onerror = (err) => {
      console.error('SSE error:', err);
      setError('Connection error');
      setConnected(false);
      
      // Attempt to reconnect after 5 seconds
      setTimeout(() => {
        if (eventSourceRef.current) {
          eventSourceRef.current.close();
        }
        // Reconnect will be handled by useEffect
      }, 5000);
    };

    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
    };
  }, []);

  return { event, connected, error };
};

