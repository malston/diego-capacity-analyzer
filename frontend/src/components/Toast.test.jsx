// ABOUTME: Tests for Toast notification component
// ABOUTME: Covers rendering, variants, dismiss behavior, and accessibility

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import Toast from './Toast';

describe('Toast', () => {
  it('renders message', () => {
    render(<Toast message="Test message" onDismiss={() => {}} />);
    expect(screen.getByText('Test message')).toBeInTheDocument();
  });

  it('renders success variant with green styling', () => {
    render(<Toast message="Success!" variant="success" onDismiss={() => {}} />);
    const toast = screen.getByRole('status');
    expect(toast).toHaveClass('bg-emerald-500/20');
  });

  it('renders error variant with red styling', () => {
    render(<Toast message="Error!" variant="error" onDismiss={() => {}} />);
    const toast = screen.getByRole('alert');
    expect(toast).toHaveClass('bg-red-500/20');
  });

  it('calls onDismiss when close button clicked', async () => {
    const onDismiss = vi.fn();
    render(<Toast message="Test" onDismiss={onDismiss} />);
    await userEvent.click(screen.getByRole('button', { name: /dismiss/i }));
    expect(onDismiss).toHaveBeenCalled();
  });

  it('has ARIA live region for accessibility', () => {
    render(<Toast message="Accessible message" onDismiss={() => {}} />);
    const toast = screen.getByRole('status');
    expect(toast).toHaveAttribute('aria-live', 'polite');
  });

  it('uses assertive aria-live for errors', () => {
    render(<Toast message="Error!" variant="error" onDismiss={() => {}} />);
    const toast = screen.getByRole('alert');
    expect(toast).toHaveAttribute('aria-live', 'assertive');
  });
});
