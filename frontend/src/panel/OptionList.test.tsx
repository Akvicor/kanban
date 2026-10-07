import {fireEvent, render, screen} from '@testing-library/react'
import {describe, expect, it, vi} from 'vitest'
import {OptionList} from './OptionList'

function renderList(overrides: Partial<Parameters<typeof OptionList>[0]> = {}) {
  const props = {
    items: [
      {id: 1, name: 'P0', color: '#E5484D'},
      {id: 2, name: 'P1', color: '#F76B15'},
    ],
    nameRequired: true,
    nameMaxLength: 20,
    defaultColor: '#8B8D98',
    placeholder: '挡位名称',
    onCreate: vi.fn(async () => {}),
    onUpdate: vi.fn(async () => {}),
    onMove: vi.fn(async () => {}),
    onDelete: vi.fn(async () => {}),
    ...overrides,
  }
  render(<OptionList {...props} />)
  return props
}

describe('OptionList', () => {
  it('名称失去焦点时保存；必填名称清空时不保存并恢复原值', () => {
    const props = renderList()
    const [first] = screen.getAllByLabelText('名称')
    fireEvent.change(first, {target: {value: ' 最高 '}})
    fireEvent.blur(first)
    expect(props.onUpdate).toHaveBeenCalledWith(1, '最高', '#E5484D')

    const [, second] = screen.getAllByLabelText('名称')
    fireEvent.change(second, {target: {value: '  '}})
    fireEvent.blur(second)
    expect(props.onUpdate).toHaveBeenCalledTimes(1)
    expect(second).toHaveValue('P1')
  })

  it('删除需要再点一次确认；第一项不能上移', () => {
    const props = renderList()
    expect(screen.getAllByRole('button', {name: '上移'})[0]).toBeDisabled()

    fireEvent.click(screen.getAllByRole('button', {name: '删除'})[1])
    expect(props.onDelete).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', {name: '确认删除'}))
    expect(props.onDelete).toHaveBeenCalledWith(2)
  })

  it('名称必填时，空名称不能添加', () => {
    const props = renderList()
    expect(screen.getByRole('button', {name: '添加'})).toBeDisabled()
    fireEvent.change(screen.getByLabelText('新建项的名称'), {target: {value: 'P4'}})
    fireEvent.click(screen.getByRole('button', {name: '添加'}))
    expect(props.onCreate).toHaveBeenCalledWith('P4', expect.stringMatching(/^#8b8d98$/i))
  })
})
