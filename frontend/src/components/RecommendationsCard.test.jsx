// ABOUTME: Tests for upgrade recommendations display component
// ABOUTME: Covers recommendation cards, priority ordering, and action display

import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import RecommendationsCard from './RecommendationsCard';

describe('RecommendationsCard', () => {
  const defaultProps = {
    recommendations: [
      {
        id: 'add-cells',
        title: 'Add Diego Cells',
        description: 'Add 2 more Diego cells to increase memory capacity',
        priority: 1,
        type: 'scale-out',
        impact: 'high',
      },
      {
        id: 'resize-cells',
        title: 'Resize Cells',
        description: 'Increase cell memory from 64GB to 128GB',
        priority: 2,
        type: 'scale-up',
        impact: 'medium',
      },
      {
        id: 'add-hosts',
        title: 'Add Hosts',
        description: 'Add 1 physical host to your cluster',
        priority: 3,
        type: 'infrastructure',
        impact: 'low',
      },
    ],
  };

  describe('Section Header', () => {
    it('renders section title', () => {
      render(<RecommendationsCard {...defaultProps} />);
      expect(screen.getByText(/recommendations/i)).toBeInTheDocument();
    });

    it('displays icon in header', () => {
      const { container } = render(<RecommendationsCard {...defaultProps} />);
      expect(container.querySelector('.lucide')).toBeInTheDocument();
    });
  });

  describe('Recommendation Cards', () => {
    it('renders all recommendations', () => {
      render(<RecommendationsCard {...defaultProps} />);

      expect(screen.getByText(/add diego cells/i)).toBeInTheDocument();
      expect(screen.getByText(/resize cells/i)).toBeInTheDocument();
      expect(screen.getByText(/add hosts/i)).toBeInTheDocument();
    });

    it('displays recommendation descriptions', () => {
      render(<RecommendationsCard {...defaultProps} />);

      expect(screen.getByText(/add 2 more diego cells/i)).toBeInTheDocument();
    });

    it('shows priority ordering numbers', () => {
      render(<RecommendationsCard {...defaultProps} />);

      expect(screen.getByText('1')).toBeInTheDocument();
      expect(screen.getByText('2')).toBeInTheDocument();
      expect(screen.getByText('3')).toBeInTheDocument();
    });
  });

  describe('Priority Ordering', () => {
    it('displays recommendations in priority order', () => {
      const { container } = render(<RecommendationsCard {...defaultProps} />);

      const cards = container.querySelectorAll('[data-recommendation]');
      expect(cards.length).toBe(3);

      // Check that first card has priority 1 recommendation
      expect(cards[0]).toHaveTextContent(/add diego cells/i);
    });

    it('handles unsorted recommendations by sorting them', () => {
      const unsortedRecommendations = [
        { id: 'low', title: 'Low Priority', priority: 3, type: 'other' },
        { id: 'high', title: 'High Priority', priority: 1, type: 'other' },
        { id: 'mid', title: 'Medium Priority', priority: 2, type: 'other' },
      ];

      const { container } = render(
        <RecommendationsCard recommendations={unsortedRecommendations} />
      );

      const cards = container.querySelectorAll('[data-recommendation]');
      expect(cards[0]).toHaveTextContent(/high priority/i);
      expect(cards[1]).toHaveTextContent(/medium priority/i);
      expect(cards[2]).toHaveTextContent(/low priority/i);
    });
  });

  describe('Impact Indicators', () => {
    it('shows high impact indicator', () => {
      render(<RecommendationsCard {...defaultProps} />);
      expect(screen.getByText(/high/i)).toBeInTheDocument();
    });

    it('shows medium impact indicator', () => {
      render(<RecommendationsCard {...defaultProps} />);
      expect(screen.getByText(/medium/i)).toBeInTheDocument();
    });

    it('shows low impact indicator', () => {
      render(<RecommendationsCard {...defaultProps} />);
      // Multiple matches for "low" - use getAllByText
      expect(screen.getAllByText(/low/i).length).toBeGreaterThan(0);
    });

    it('uses color coding for impact levels', () => {
      const { container } = render(<RecommendationsCard {...defaultProps} />);

      // High impact should have emerald color
      expect(container.querySelector('.text-emerald-400')).toBeInTheDocument();
      // Medium impact should have amber color
      expect(container.querySelector('.text-amber-400')).toBeInTheDocument();
    });
  });

  describe('Recommendation Types', () => {
    it('displays type badges for scale-out', () => {
      render(<RecommendationsCard {...defaultProps} />);
      expect(screen.getByText(/scale-out/i)).toBeInTheDocument();
    });

    it('displays type badges for scale-up', () => {
      render(<RecommendationsCard {...defaultProps} />);
      expect(screen.getByText(/scale-up/i)).toBeInTheDocument();
    });

    it('displays type badges for infrastructure', () => {
      render(<RecommendationsCard {...defaultProps} />);
      expect(screen.getByText(/infrastructure/i)).toBeInTheDocument();
    });
  });

  describe('Edge Cases', () => {
    it('handles empty recommendations array', () => {
      render(<RecommendationsCard recommendations={[]} />);
      expect(screen.getByText(/no recommendations/i)).toBeInTheDocument();
    });

    it('handles single recommendation', () => {
      render(
        <RecommendationsCard
          recommendations={[
            { id: 'single', title: 'Single Rec', priority: 1, type: 'other' },
          ]}
        />
      );
      expect(screen.getByText(/single rec/i)).toBeInTheDocument();
    });

    it('handles missing optional fields gracefully', () => {
      render(
        <RecommendationsCard
          recommendations={[
            { id: 'minimal', title: 'Minimal', priority: 1, type: 'other' },
          ]}
        />
      );
      expect(screen.getByText(/minimal/i)).toBeInTheDocument();
    });
  });

  describe('Icons', () => {
    it('displays appropriate icons for recommendation types', () => {
      const { container } = render(<RecommendationsCard {...defaultProps} />);

      // Should have lucide icons
      expect(container.querySelector('.lucide-plus')).toBeInTheDocument();
      expect(container.querySelector('.lucide-arrow-up')).toBeInTheDocument();
      expect(container.querySelector('.lucide-server')).toBeInTheDocument();
    });
  });
});
