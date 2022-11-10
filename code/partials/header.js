import PropTypes from 'prop-types';
import React, { Fragment } from 'react';

import Nav from './nav';

/**
 * The Header component
 *
 * @disable-docs
 */
const Header = ({ title, _parents, _ID, _pages, _nav, _globalProp }) => (
	<Fragment>
		<Nav _ID={_ID} _pages={_pages} _nav={_nav} />
		<div className="wrapper">
			<div className="section started smallheader">
				<div className="centrize full-width">
					<div className="started-content">
						<div className="h-title glitch-effect" data-text={ _globalProp['sitename'] }>{ _globalProp['sitename'] }</div>
						<div className="h-subtitle typing-bread">
							<p>
							{
								_parents.slice(1)
									.map(
										(page, i) =>(
										<Fragment key={i}>
											{ i > 0 ? ' / ' : null }
											<a href={_pages[page]._url}>{_pages[page].title}</a>
										</Fragment>
										)
									)
							}
							</p>
						</div>
						<span className="typed-bread"></span>
					</div>
				</div>
			</div>
		</div>
	</Fragment>
);

Header.defaultProps = {};

export default Header;
