import PropTypes from 'prop-types';
import React, { Fragment } from 'react';

/**
 * The Header component
 *
 * @disable-docs
 */
const Header = ({ title, _parents, _pages, _globalProp }) => (
	<Fragment>
		<div className="section started">
			<div className="centrize full-width">
				<div className="vertical-center">
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
			<a href="#" className="mouse_btn"><span className="ion ion-mouse"></span></a>
		</div>
	</Fragment>
);

Header.defaultProps = {};

export default Header;
