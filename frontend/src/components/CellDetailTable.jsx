// ABOUTME: Detailed Diego cell table with capacity and utilization metrics
// ABOUTME: Shows per-cell memory allocation, usage, CPU, and visual utilization bars

import { Server } from 'lucide-react';

const CellDetailTable = ({ cells, selectedSegment }) => {
  const filteredCells = selectedSegment === 'all'
    ? cells
    : cells.filter(c => c.isolation_segment === selectedSegment);

  return (
    <div className="metric-card p-6 rounded-xl mb-8">
      <h2 className="text-lg font-bold title-font text-white mb-4 flex items-center gap-2">
        <Server className="w-5 h-5 text-blue-400" aria-hidden="true" />
        Diego Cells Detail
      </h2>
      <div className="overflow-x-auto">
        <table className="w-full text-sm" role="table" aria-label="Diego cell details">
          <thead>
            <tr className="border-b border-slate-700 text-slate-400">
              <th scope="col" className="text-left py-3 px-4">Cell</th>
              <th scope="col" className="text-left py-3 px-4">Segment</th>
              <th scope="col" className="text-right py-3 px-4">Capacity</th>
              <th scope="col" className="text-right py-3 px-4">Allocated</th>
              <th scope="col" className="text-right py-3 px-4">Used</th>
              <th scope="col" className="text-right py-3 px-4">CPU</th>
              <th scope="col" className="text-left py-3 px-4">Utilization</th>
            </tr>
          </thead>
          <tbody>
            {filteredCells.map((cell) => {
              const utilizationPercent = (cell.used_mb / cell.memory_mb) * 100;
              const status = utilizationPercent > 80 ? 'high' : utilizationPercent > 60 ? 'medium' : 'low';

              return (
                <tr key={cell.id} className="cell-row border-b border-slate-800">
                  <td className="py-3 px-4 font-semibold text-white">{cell.name}</td>
                  <td className="py-3 px-4">
                    <span className="segment-chip">{cell.isolation_segment}</span>
                  </td>
                  <td className="py-3 px-4 text-right text-slate-300">{cell.memory_mb} MB</td>
                  <td className="py-3 px-4 text-right text-slate-300">{cell.allocated_mb} MB</td>
                  <td className="py-3 px-4 text-right text-white font-semibold">{cell.used_mb} MB</td>
                  <td className="py-3 px-4 text-right">
                    <span
                      className={`font-semibold ${
                        cell.cpu_percent > 70 ? 'text-red-400' :
                        cell.cpu_percent > 50 ? 'text-amber-400' :
                        'text-emerald-400'
                      }`}
                    >
                      {cell.cpu_percent}%
                    </span>
                  </td>
                  <td className="py-3 px-4">
                    <div className="flex items-center gap-3">
                      <div
                        className="progress-bar w-32 h-2 rounded-full"
                        role="progressbar"
                        aria-valuenow={utilizationPercent}
                        aria-valuemin={0}
                        aria-valuemax={100}
                        aria-label={`${cell.name} memory utilization`}
                      >
                        <div
                          className={`progress-fill h-full rounded-full ${
                            status === 'high' ? 'bg-red-500' :
                            status === 'medium' ? 'bg-amber-500' :
                            'bg-emerald-500'
                          }`}
                          style={{ width: `${utilizationPercent}%` }}
                        />
                      </div>
                      <span className="text-slate-300 font-semibold">{utilizationPercent.toFixed(1)}%</span>
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default CellDetailTable;
